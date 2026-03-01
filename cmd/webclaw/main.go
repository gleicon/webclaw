//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/agent"
	"github.com/gleicon/webclaw/internal/config"
	"github.com/gleicon/webclaw/internal/identity"
	"github.com/gleicon/webclaw/internal/jsbridge"
	"github.com/gleicon/webclaw/internal/keystore"
	"github.com/gleicon/webclaw/internal/provider"
	"github.com/gleicon/webclaw/internal/tools"
)

func main() {
	jsbridge.Init()

	// Register export/import bridge
	registerExportImportBridge()

	// Register identity file JS bridge (webclaw.identity.getFile/putFile/listFiles)
	// Must be called before InitWorkerBridge so the webclaw object exists
	registerIdentityBridge()

	// Register API key management JS bridge (webclaw.keystore.setKey/hasKey)
	// Must be called before InitWorkerBridge so the webclaw object exists
	registerKeystoreBridge()

	// Initialize configuration
	if err := initializeConfig(); err != nil {
		js.Global().Get("console").Call("error", "webclaw: config initialization failed:", err.Error())
		// Don't exit - we can still run without config for now
	}

	// Initialize keystore
	if err := initializeKeystore(); err != nil {
		js.Global().Get("console").Call("error", "webclaw: keystore initialization failed:", err.Error())
		// Don't exit - we can still run without keystore for now
	}

	// Initialize identity files
	if err := initializeIdentity(); err != nil {
		js.Global().Get("console").Call("error", "webclaw: identity initialization failed:", err.Error())
		// Don't exit - we can still run without identity for now
	}

	// Initialize worker bridge for streaming.
	// InitWorkerBridge returns the *WorkerBridge instance for wiring into AgentLoop.
	workerBridgeInstance := agent.InitWorkerBridge()

	// Create the persistent AgentLoop (no provider/model yet; set via webclaw.keystore.setKey at runtime)
	agentLoop := agent.NewAgentLoop("", "")

	// Wire real provider router so getProvider() returns a real LLM, not mockProvider.
	// API keys start empty; user sets them via webclaw.keystore.setKey in the Settings tab.
	// TODO v2: load persisted keys from keystore at startup (requires async init).
	routerConfig := &provider.Config{
		HTTPReferer: "https://github.com/gleicon/webclaw",
		XTitle:      "WebClaw",
	}
	router := provider.NewRouter(routerConfig)
	agentLoop.SetRouter(router)

	// Wire tool registry with all four browser tools.
	// Without this call, toolRegistry == nil and every tool call returns "tool registry not configured".
	reg := tools.NewRegistry()
	reg.Register(tools.NewWebFetchTool())
	reg.Register(tools.NewWebSearchTool())
	reg.Register(tools.NewMemoryStoreTool(agentLoop))
	reg.Register(tools.NewMemorySearchTool(agentLoop))
	agentLoop.SetToolRegistry(reg)

	// Wire worker bridge so EmitToolEvent calls from the dispatch loop reach the UI.
	agentLoop.SetWorkerBridge(workerBridgeInstance)

	// Register the pre-configured agentLoop for use by handleStartStream.
	agent.SetGlobalAgentLoop(agentLoop)

	js.Global().Get("console").Call("log", "webclaw: export/import ready")
	js.Global().Get("console").Call("log", "webclaw: WASM ready")
	<-make(chan struct{}) // block forever — Go runtime exits when main() returns
}

func initializeKeystore() error {
	ks, err := keystore.NewKeyStore()
	if err != nil {
		return fmt.Errorf("failed to create keystore: %w", err)
	}

	// Just verify it works - the keystore reference is managed internally
	_ = ks
	js.Global().Get("console").Call("log", "webclaw: keystore initialized")
	return nil
}

func initializeConfig() error {
	storage, err := config.NewStorage()
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}
	defer storage.Close()

	// Check if config exists
	exists, err := storage.ConfigExists()
	if err != nil {
		return fmt.Errorf("failed to check config existence: %w", err)
	}

	if !exists {
		// First run - create default config
		cfg := config.DefaultConfig()
		if err := storage.SetConfig(cfg); err != nil {
			return fmt.Errorf("failed to save default config: %w", err)
		}
		// Dispatch event with config version and identity name only
		// Full config can be retrieved via storage API
		js.Global().Call("dispatchEvent",
			js.Global().Get("CustomEvent").New("webclaw:first-run",
				map[string]interface{}{
					"version":  cfg.Version,
					"identity": cfg.Identity.Name,
				}))
		js.Global().Get("console").Call("log", "webclaw: created default config (first run)")
	} else {
		// Config exists - load it
		cfg, err := storage.GetConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		// Dispatch event with basic info
		js.Global().Call("dispatchEvent",
			js.Global().Get("CustomEvent").New("webclaw:config-ready",
				map[string]interface{}{
					"version":  cfg.Version,
					"identity": cfg.Identity.Name,
				}))
		js.Global().Get("console").Call("log", "webclaw: config loaded")
	}

	return nil
}

func initializeIdentity() error {
	store, err := identity.NewStore()
	if err != nil {
		return fmt.Errorf("failed to create identity store: %w", err)
	}
	defer store.Close()

	// Check if any identity files exist
	files, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list identity files: %w", err)
	}

	if len(files) == 0 {
		// First run - create default identity files
		if err := store.LoadDefaults(); err != nil {
			return fmt.Errorf("failed to load default identity files: %w", err)
		}

		js.Global().Call("dispatchEvent",
			js.Global().Get("CustomEvent").New("webclaw:identity-ready",
				map[string]interface{}{
					"filesCreated": 6,
					"event":        "first-run",
				}))
		js.Global().Get("console").Call("log", "webclaw: created default identity files (first run)")
	} else {
		// Identity files exist
		js.Global().Call("dispatchEvent",
			js.Global().Get("CustomEvent").New("webclaw:identity-ready",
				map[string]interface{}{
					"filesLoaded": len(files),
					"event":       "loaded",
				}))
		js.Global().Get("console").Call("log", "webclaw: identity files loaded")
	}

	return nil
}

// registerExportImportBridge registers the export/import JavaScript bridge
func registerExportImportBridge() {
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		return // jsbridge.Init() not called yet
	}

	exportImport := js.Global().Get("Object").New()

	// Export function: webclaw.exportImport.exportConfig()
	exportFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				// Get config and export
				storage, err := config.NewStorage()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				defer storage.Close()

				cfg, err := storage.GetConfig()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				idStore, err := identity.NewStore()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				defer idStore.Close()

				// Create identity provider wrapper
				idProvider := &identityFileProvider{store: idStore}

				data, err := config.ExportAll(cfg, idProvider, nil) // No keystore for now
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				jsonBytes, err := config.ExportToJSON(data)
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				// Trigger download
				jsbridge.TriggerDownload("webclaw-config.json", jsonBytes)
				resolve.Invoke(js.Undefined())
			}()

			return nil
		}))
	})
	jsbridge.RegisterCallback(exportFn)
	exportImport.Set("exportConfig", exportFn)

	// Import function: webclaw.exportImport.importConfig(jsonContent)
	importFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) < 1 {
			return js.Undefined()
		}
		jsonContent := args[0].String()

		promiseCtor := js.Global().Get("Promise")
		return promiseCtor.New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]

			go func() {
				data, err := config.ImportFromJSON([]byte(jsonContent))
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				storage, err := config.NewStorage()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				defer storage.Close()

				idStore, err := identity.NewStore()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				defer idStore.Close()

				// Create identity importer wrapper
				idImporter := &identityFileImporter{store: idStore}

				if err := config.ImportAll(data, storage, idImporter); err != nil {
					reject.Invoke(err.Error())
					return
				}

				resolve.Invoke(js.Undefined())
			}()

			return nil
		}))
	})
	jsbridge.RegisterCallback(importFn)
	exportImport.Set("importConfig", importFn)

	webclaw.Set("exportImport", exportImport)
}

// registerIdentityBridge registers the identity file JS bridge.
// Provides webclaw.identity.getFile(filename), putFile(filename, content), listFiles() — all return Promises.
// Each function opens and closes a fresh store connection per call.
// Called before agent.InitWorkerBridge() so the webclaw global object exists.
func registerIdentityBridge() {
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		js.Global().Get("console").Call("warn", "webclaw: identity bridge: webclaw object not found")
		return
	}

	obj := js.Global().Get("Object").New()

	// webclaw.identity.getFile(filename) Promise<string>
	getFileFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		filename := ""
		if len(args) > 0 {
			filename = args[0].String()
		}
		return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]
			go func() {
				store, err := identity.NewStore()
				if err != nil {
					reject.Invoke("failed to open identity store: " + err.Error())
					return
				}
				defer store.Close()
				file, err := store.Get(filename)
				if err != nil {
					reject.Invoke("failed to get file: " + err.Error())
					return
				}
				if file == nil {
					resolve.Invoke("")
					return
				}
				resolve.Invoke(file.Content)
			}()
			return nil
		}))
	})
	jsbridge.RegisterCallback(getFileFn)
	obj.Set("getFile", getFileFn)

	// webclaw.identity.putFile(filename, content) Promise<void>
	putFileFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		filename := ""
		content := ""
		if len(args) > 0 {
			filename = args[0].String()
		}
		if len(args) > 1 {
			content = args[1].String()
		}
		return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]
			go func() {
				store, err := identity.NewStore()
				if err != nil {
					reject.Invoke("failed to open identity store: " + err.Error())
					return
				}
				defer store.Close()
				file := &identity.IdentityFile{
					Filename: filename,
					Content:  content,
				}
				if err := store.Put(file); err != nil {
					reject.Invoke("failed to put file: " + err.Error())
					return
				}
				resolve.Invoke(js.Undefined())
			}()
			return nil
		}))
	})
	jsbridge.RegisterCallback(putFileFn)
	obj.Set("putFile", putFileFn)

	// webclaw.identity.listFiles() Promise<string[]>
	listFilesFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]
			go func() {
				store, err := identity.NewStore()
				if err != nil {
					reject.Invoke("failed to open identity store: " + err.Error())
					return
				}
				defer store.Close()
				filenames, err := store.List()
				if err != nil {
					reject.Invoke("failed to list files: " + err.Error())
					return
				}
				arr := js.Global().Get("Array").New()
				for _, name := range filenames {
					arr.Call("push", name)
				}
				resolve.Invoke(arr)
			}()
			return nil
		}))
	})
	jsbridge.RegisterCallback(listFilesFn)
	obj.Set("listFiles", listFilesFn)

	webclaw.Set("identity", obj)
	js.Global().Get("console").Call("log", "webclaw: identity bridge registered")
}

// registerKeystoreBridge registers the API key management JS bridge.
// Provides webclaw.keystore.setKey(provider, apiKey) and hasKey(provider) — both return Promises.
// Called before agent.InitWorkerBridge() so the webclaw global object exists.
func registerKeystoreBridge() {
	webclaw := js.Global().Get("webclaw")
	if webclaw.IsUndefined() || webclaw.IsNull() {
		js.Global().Get("console").Call("warn", "webclaw: keystore bridge: webclaw object not found")
		return
	}

	obj := js.Global().Get("Object").New()

	// webclaw.keystore.setKey(provider, apiKey) Promise<void>
	// TODO v2: derive passphrase from user input (prompt or secure storage)
	setKeyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		providerName := ""
		apiKey := ""
		if len(args) > 0 {
			providerName = args[0].String()
		}
		if len(args) > 1 {
			apiKey = args[1].String()
		}
		return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]
			go func() {
				ks, err := keystore.NewKeyStore()
				if err != nil {
					reject.Invoke("failed to open keystore: " + err.Error())
					return
				}
				// v1 simplification: fixed passphrase. Keys are still encrypted at rest.
				// TODO v2: derive passphrase from user input
				const passphrase = "webclaw-v1-key"
				if err := ks.StoreKey(providerName, apiKey, passphrase); err != nil {
					reject.Invoke(fmt.Sprintf("failed to store key for %s: %s", providerName, err.Error()))
					return
				}
				resolve.Invoke(js.Undefined())
			}()
			return nil
		}))
	})
	jsbridge.RegisterCallback(setKeyFn)
	obj.Set("setKey", setKeyFn)

	// webclaw.keystore.hasKey(provider) Promise<bool>
	hasKeyFn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		providerName := ""
		if len(args) > 0 {
			providerName = args[0].String()
		}
		return js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, resolveReject []js.Value) interface{} {
			resolve := resolveReject[0]
			reject := resolveReject[1]
			go func() {
				ks, err := keystore.NewKeyStore()
				if err != nil {
					reject.Invoke("failed to open keystore: " + err.Error())
					return
				}
				exists, err := ks.KeyExists(providerName)
				if err != nil {
					reject.Invoke(fmt.Sprintf("failed to check key for %s: %s", providerName, err.Error()))
					return
				}
				resolve.Invoke(js.ValueOf(exists))
			}()
			return nil
		}))
	})
	jsbridge.RegisterCallback(hasKeyFn)
	obj.Set("hasKey", hasKeyFn)

	webclaw.Set("keystore", obj)
	js.Global().Get("console").Call("log", "webclaw: keystore bridge registered")
}

// identityFileProvider wraps identity.Store to implement config.IdentityFileProvider
type identityFileProvider struct {
	store *identity.Store
}

func (p *identityFileProvider) List() ([]string, error) {
	return p.store.List()
}

func (p *identityFileProvider) GetContent(filename string) (string, error) {
	file, err := p.store.Get(filename)
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", nil
	}
	return file.Content, nil
}

// identityFileImporter wraps identity.Store to implement config.IdentityFileImporter
type identityFileImporter struct {
	store *identity.Store
}

func (i *identityFileImporter) PutContent(filename string, content string) error {
	file := &identity.IdentityFile{
		Filename: filename,
		Content:  content,
	}
	return i.store.Put(file)
}
