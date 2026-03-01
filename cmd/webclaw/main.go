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

// globalRouter holds the provider router for access by JS bridge functions.
// This allows setKey to register providers immediately when keys are added.
var globalRouter *provider.Router

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
	globalRouter = router // Store for access by JS bridge functions
	agentLoop.SetRouter(router)

	// Load persisted API keys from keystore asynchronously (Wave 1: async initialization)
	go loadProviderKeysAsync(router)

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

				// If no config exists, create a default one
				if cfg == nil {
					cfg = config.DefaultConfig()
				}

				idStore, err := identity.NewStore()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}
				defer idStore.Close()

				// Create identity provider wrapper
				idProvider := &identityFileProvider{store: idStore}

				// Get keystore for exporting encrypted keys
				ks, err := keystore.NewKeyStore()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				data, err := config.ExportAll(cfg, idProvider, ks)
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

				// Create keystore for importing API keys
				ks, err := keystore.NewKeyStore()
				if err != nil {
					reject.Invoke(err.Error())
					return
				}

				if err := config.ImportAll(data, storage, idImporter, ks); err != nil {
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
				// Register the provider immediately so it's available without page reload
				registerProviderAndNotify(providerName, apiKey)
				// Clear key from memory after registration (best effort security)
				keystore.ClearKey(apiKey)
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

// registerProviderAndNotify creates a provider instance from an API key,
// registers it with the global router, and dispatches the providers-ready event.
// Used by setKey to make newly-added keys immediately available without page reload.
func registerProviderAndNotify(providerName, apiKey string) {
	if globalRouter == nil {
		js.Global().Get("console").Call("error", "webclaw: cannot register provider, router not initialized")
		return
	}

	// Create the appropriate provider instance
	var providerInstance provider.Provider
	switch providerName {
	case "anthropic":
		providerInstance = provider.NewAnthropicProvider(apiKey)
	case "openai":
		providerInstance = provider.NewOpenAIProvider(apiKey)
	case "openrouter":
		providerInstance = provider.NewOpenRouterProvider(apiKey, "https://github.com/gleicon/webclaw", "WebClaw")
	}

	if providerInstance != nil {
		globalRouter.RegisterProvider(providerName, providerInstance)
		js.Global().Get("console").Call("log", "webclaw: registered provider:", providerName)
	}

	// Dispatch event with updated provider list
	availableProviders := globalRouter.AvailableProviders()
	js.Global().Call("dispatchEvent",
		js.Global().Get("CustomEvent").New("webclaw:providers-ready",
			map[string]interface{}{
				"providers": availableProviders,
				"count":     len(availableProviders),
			}))

	js.Global().Get("console").Call("log", "webclaw: providers ready, count:", len(availableProviders))
}

// loadProviderKeysAsync asynchronously loads persisted API keys from the keystore
// and registers them with the provider router. This runs in a goroutine to avoid
// blocking the main thread during IndexedDB operations.
func loadProviderKeysAsync(router *provider.Router) {
	// Fixed passphrase for v1 keystore encryption
	const passphrase = "webclaw-v1-key"

	// Open keystore connection to IndexedDB
	ks, err := keystore.NewKeyStore()
	if err != nil {
		js.Global().Get("console").Call("warn", "webclaw: keystore open failed, no persisted keys loaded:", err.Error())
		return
	}

	// List of providers to attempt loading
	providers := []string{"anthropic", "openai", "openrouter"}

	// Iterate through providers, loading each key if it exists
	for _, providerName := range providers {
		// Check if key exists before attempting retrieval
		exists, err := ks.KeyExists(providerName)
		if err != nil {
			js.Global().Get("console").Call("error", "webclaw: failed to check key existence for", providerName+":", err.Error())
			continue
		}

		if !exists {
			js.Global().Get("console").Call("log", "webclaw: no persisted key for", providerName)
			continue
		}

		// Retrieve and decrypt the API key
		apiKey, err := ks.RetrieveKey(providerName, passphrase)
		if err != nil {
			js.Global().Get("console").Call("error", "webclaw: failed to retrieve key for", providerName+":", err.Error())
			continue
		}

		// Register the provider with the router based on provider name
		var providerInstance provider.Provider
		switch providerName {
		case "anthropic":
			providerInstance = provider.NewAnthropicProvider(apiKey)
		case "openai":
			providerInstance = provider.NewOpenAIProvider(apiKey)
		case "openrouter":
			providerInstance = provider.NewOpenRouterProvider(apiKey, "https://github.com/gleicon/webclaw", "WebClaw")
		}

		if providerInstance != nil {
			router.RegisterProvider(providerName, providerInstance)
			js.Global().Get("console").Call("log", "webclaw: loaded persisted key for", providerName)
		}

		// Clear key from memory (best effort security)
		keystore.ClearKey(apiKey)
	}

	// Dispatch event with available provider list to notify UI
	availableProviders := router.AvailableProviders()
	js.Global().Call("dispatchEvent",
		js.Global().Get("CustomEvent").New("webclaw:providers-ready",
			map[string]interface{}{
				"providers": availableProviders,
				"count":     len(availableProviders),
			}))

	js.Global().Get("console").Call("log", "webclaw: async keystore initialization complete, providers:", len(availableProviders))
}
