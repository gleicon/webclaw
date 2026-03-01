//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/gleicon/webclaw/internal/config"
	"github.com/gleicon/webclaw/internal/jsbridge"
	"github.com/gleicon/webclaw/internal/keystore"
)

func main() {
	jsbridge.Init()

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
