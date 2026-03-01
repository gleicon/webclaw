//go:build js && wasm

package jsbridge

import (
	"errors"
	"syscall/js"
	"time"
)

// FetchResponse represents the result of a fetch call
type FetchResponse struct {
	Status     int
	StatusText string
	Headers    map[string]string
	Body       []byte
}

// FetchOptions contains options for fetch calls
type FetchOptions struct {
	Method  string
	Headers map[string]string
	Body    string
}

// Fetch makes an HTTP request using syscall/js fetch()
// This is the primary HTTP mechanism for the provider package - no net/http allowed
func Fetch(url string, opts FetchOptions) (*FetchResponse, error) {
	// Create the options object for JS fetch
	jsOpts := js.Global().Get("Object").New()

	if opts.Method != "" {
		jsOpts.Set("method", opts.Method)
	} else {
		jsOpts.Set("method", "GET")
	}

	// Set headers
	if len(opts.Headers) > 0 {
		headers := js.Global().Get("Object").New()
		for k, v := range opts.Headers {
			headers.Set(k, v)
		}
		jsOpts.Set("headers", headers)
	}

	// Set body
	if opts.Body != "" {
		jsOpts.Set("body", opts.Body)
	}

	// Make the fetch call synchronously through a channel
	resultChan := make(chan fetchResult, 1)

	go func() {
		// Call window.fetch
		promise := js.Global().Call("fetch", url, jsOpts)

		promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]

			// Extract status
			status := response.Get("status").Int()
			statusText := response.Get("statusText").String()

			// Extract headers
			headers := make(map[string]string)
			headersObj := response.Get("headers")
			headersIter := headersObj.Call("entries")
			for {
				next := headersIter.Call("next")
				done := next.Get("done").Bool()
				if done {
					break
				}
				entry := next.Get("value")
				key := entry.Index(0).String()
				value := entry.Index(1).String()
				headers[key] = value
			}

			// Read the body as text
			bodyPromise := response.Call("text")
			bodyPromise.Call("then", js.FuncOf(func(this js.Value, textArgs []js.Value) interface{} {
				bodyText := textArgs[0].String()
				resultChan <- fetchResult{
					resp: &FetchResponse{
						Status:     status,
						StatusText: statusText,
						Headers:    headers,
						Body:       []byte(bodyText),
					},
					err: nil,
				}
				return nil
			})).Call("catch", js.FuncOf(func(this js.Value, errArgs []js.Value) interface{} {
				errMsg := errArgs[0].Get("message").String()
				resultChan <- fetchResult{
					resp: &FetchResponse{
						Status:     status,
						StatusText: statusText,
						Headers:    headers,
						Body:       []byte{},
					},
					err: errors.New(errMsg),
				}
				return nil
			}))
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errMsg := args[0].Get("message").String()
			resultChan <- fetchResult{
				resp: nil,
				err:  errors.New("fetch failed: " + errMsg),
			}
			return nil
		}))
	}()

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		return result.resp, result.err
	case <-time.After(30 * time.Second):
		return nil, errors.New("fetch timeout after 30s")
	}
}

type fetchResult struct {
	resp *FetchResponse
	err  error
}

// FetchStream initiates a streaming fetch request
// Returns the response object and a function to get the reader
// Use this for SSE streaming from LLM providers
func FetchStream(url string, opts FetchOptions) (js.Value, error) {
	resultChan := make(chan streamResult, 1)

	go func() {
		// Create options
		jsOpts := js.Global().Get("Object").New()

		if opts.Method != "" {
			jsOpts.Set("method", opts.Method)
		} else {
			jsOpts.Set("method", "POST")
		}

		// Set headers
		if len(opts.Headers) > 0 {
			headers := js.Global().Get("Object").New()
			for k, v := range opts.Headers {
				headers.Set(k, v)
			}
			jsOpts.Set("headers", headers)
		}

		// Set body
		if opts.Body != "" {
			jsOpts.Set("body", opts.Body)
		}

		// Call fetch
		promise := js.Global().Call("fetch", url, jsOpts)

		promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]
			resultChan <- streamResult{
				response: response,
				err:      nil,
			}
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			errMsg := args[0].Get("message").String()
			resultChan <- streamResult{
				response: js.Undefined(),
				err:      errors.New("fetch stream failed: " + errMsg),
			}
			return nil
		}))
	}()

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		return result.response, result.err
	case <-time.After(30 * time.Second):
		return js.Undefined(), errors.New("fetch stream timeout after 30s")
	}
}

type streamResult struct {
	response js.Value
	err      error
}

// RegisterFetchCallback registers the enhanced fetch function on window.webclaw.jsFetch
// This replaces the simple version with one that supports options
func RegisterFetchCallback() js.Func {
	fetchFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 2 {
			return js.Undefined()
		}

		url := args[0].String()
		optionsObj := args[1]

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				opts := FetchOptions{
					Method:  optionsObj.Get("method").String(),
					Body:    optionsObj.Get("body").String(),
					Headers: make(map[string]string),
				}

				// Extract headers from JS object
				headersObj := optionsObj.Get("headers")
				if !headersObj.IsUndefined() && !headersObj.IsNull() {
					headersIter := headersObj.Call("entries")
					for {
						next := headersIter.Call("next")
						done := next.Get("done").Bool()
						if done {
							break
						}
						entry := next.Get("value")
						key := entry.Index(0).String()
						value := entry.Index(1).String()
						opts.Headers[key] = value
					}
				}

				resp, err := Fetch(url, opts)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				// Convert response to JS object
				jsResp := js.Global().Get("Object").New()
				jsResp.Set("status", resp.Status)
				jsResp.Set("statusText", resp.StatusText)
				jsResp.Set("body", string(resp.Body))

				// Convert headers
				jsHeaders := js.Global().Get("Object").New()
				for k, v := range resp.Headers {
					jsHeaders.Set(k, v)
				}
				jsResp.Set("headers", jsHeaders)

				resolve.Invoke(jsResp)
			}()

			return nil
		}))
	})

	return fetchFn
}
