//go:build js && wasm

package jsbridge

import (
	"errors"
	"syscall/js"
	"time"
)

// StreamChunk represents a chunk of data from a streaming response
type StreamChunk struct {
	Data []byte
	Done bool
	Err  error
}

// StreamingReader provides a Go-channel based interface for reading
// from a JavaScript ReadableStream (used for SSE streaming)
type StreamingReader struct {
	reader    js.Value
	chunks    chan StreamChunk
	cancelled bool
}

// NewStreamingReader creates a reader from a JS ReadableStreamDefaultReader
func NewStreamingReader(response js.Value) *StreamingReader {
	// Get the reader from the body
	body := response.Get("body")
	reader := body.Call("getReader")

	sr := &StreamingReader{
		reader: reader,
		chunks: make(chan StreamChunk, 10),
	}

	// Start reading loop
	go sr.readLoop()

	return sr
}

// readLoop continuously reads from the JS reader and sends to Go channel
func (sr *StreamingReader) readLoop() {
	defer close(sr.chunks)

	for !sr.cancelled {
		resultChan := make(chan readResult, 1)

		// Call read() on the JS reader
		go func() {
			promise := sr.reader.Call("read")
			promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				result := args[0]
				done := result.Get("done").Bool()

				if done {
					resultChan <- readResult{done: true}
				} else {
					value := result.Get("value")
					// Convert Uint8Array to Go bytes
					length := value.Get("length").Int()
					data := make([]byte, length)
					js.CopyBytesToGo(data, value)
					resultChan <- readResult{data: data, done: false}
				}
				return nil
			})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				errMsg := args[0].Get("message").String()
				resultChan <- readResult{err: errors.New(errMsg)}
				return nil
			}))
		}()

		// Wait for read with timeout
		select {
		case result := <-resultChan:
			if result.err != nil {
				sr.chunks <- StreamChunk{Err: result.err}
				return
			}
			if result.done {
				sr.chunks <- StreamChunk{Done: true}
				return
			}
			sr.chunks <- StreamChunk{Data: result.data}

		case <-time.After(60 * time.Second):
			sr.chunks <- StreamChunk{Err: errors.New("stream read timeout after 60s")}
			return
		}
	}
}

type readResult struct {
	data []byte
	done bool
	err  error
}

// ReadChunks returns the channel for receiving stream chunks
func (sr *StreamingReader) ReadChunks() <-chan StreamChunk {
	return sr.chunks
}

// Cancel stops the streaming reader
func (sr *StreamingReader) Cancel() {
	sr.cancelled = true
	// Cancel the JS reader if possible
	if !sr.reader.IsUndefined() && !sr.reader.IsNull() {
		sr.reader.Call("cancel")
	}
}

// SSEStreamingReader wraps StreamingReader with SSE parsing
type SSEStreamingReader struct {
	streamer *StreamingReader
	parser   *SSEParser
}

// SSEParser parses Server-Sent Events from a byte stream
type SSEParser struct {
	buffer []byte
	done   bool
}

// NewSSEParser creates a new SSE parser
func NewSSEParser() *SSEParser {
	return &SSEParser{
		buffer: make([]byte, 0, 4096),
	}
}

// Write adds data to the parser buffer
func (p *SSEParser) Write(data []byte) {
	p.buffer = append(p.buffer, data...)
}

// MarkDone signals that the stream is complete
func (p *SSEParser) MarkDone() {
	p.done = true
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Event string
	Data  string
	ID    string
	Retry int
}

// NextEvent returns the next SSE event or nil if not available
func (p *SSEParser) NextEvent() *SSEEvent {
	data := string(p.buffer)

	// Look for double newline (event terminator)
	var endIdx int
	found := false

	for i := 0; i < len(data)-1; i++ {
		if data[i] == '\n' && data[i+1] == '\n' {
			endIdx = i + 2
			found = true
			break
		}
		if i < len(data)-3 && data[i] == '\r' && data[i+1] == '\n' && data[i+2] == '\r' && data[i+3] == '\n' {
			endIdx = i + 4
			found = true
			break
		}
	}

	if !found {
		if p.done && len(data) > 0 {
			// Process remaining data as last event
			endIdx = len(data)
		} else {
			return nil
		}
	}

	// Parse the event block
	block := data[:endIdx]
	if found {
		p.buffer = p.buffer[endIdx:]
	} else {
		p.buffer = p.buffer[:0]
	}

	return parseSSEBlock(block)
}

func parseSSEBlock(block string) *SSEEvent {
	event := &SSEEvent{}
	lines := splitSSELines(block)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// SSE format: "field: value" or "field"
		if idx := findSSEColon(line); idx >= 0 {
			field := line[:idx]
			value := ""
			if idx+1 < len(line) {
				if line[idx+1] == ' ' {
					value = line[idx+2:]
				} else {
					value = line[idx+1:]
				}
			}

			switch field {
			case "event":
				event.Event = value
			case "data":
				if event.Data != "" {
					event.Data += "\n"
				}
				event.Data += value
			case "id":
				event.ID = value
			case "retry":
				// Parse retry value
			}
		} else if line == "" {
			// Empty line marks end of event
		}
	}

	return event
}

func splitSSELines(s string) []string {
	var lines []string
	var start int

	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}

	if start < len(s) {
		lines = append(lines, s[start:])
	}

	return lines
}

func findSSEColon(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}

// NewSSEStreamingReader creates an SSE reader from a JS response
func NewSSEStreamingReader(response js.Value) *SSEStreamingReader {
	return &SSEStreamingReader{
		streamer: NewStreamingReader(response),
		parser:   NewSSEParser(),
	}
}

// Events returns a channel of parsed SSE events
func (sr *SSEStreamingReader) Events() <-chan *SSEEvent {
	result := make(chan *SSEEvent, 10)

	go func() {
		defer close(result)

		for chunk := range sr.streamer.ReadChunks() {
			if chunk.Err != nil {
				// Send error as event
				result <- &SSEEvent{Event: "error", Data: chunk.Err.Error()}
				return
			}

			if chunk.Done {
				sr.parser.MarkDone()
				// Process any remaining events
				for {
					event := sr.parser.NextEvent()
					if event == nil {
						break
					}
					result <- event
				}
				return
			}

			sr.parser.Write(chunk.Data)

			// Extract all available events
			for {
				event := sr.parser.NextEvent()
				if event == nil {
					break
				}
				result <- event
			}
		}
	}()

	return result
}

// Cancel stops the SSE reader
func (sr *SSEStreamingReader) Cancel() {
	sr.streamer.Cancel()
}
