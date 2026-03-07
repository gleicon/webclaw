package main

import (
	"fmt"
	"log"
	"os"
)

const (
	version     = "1.0.0"
	defaultPort = "18800"
)

func main() {
	// Check for help/version flags
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v":
			fmt.Printf("webclaw-bridge version %s\n", version)
			os.Exit(0)
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	// Get port from env or use default
	port := os.Getenv("WEBCLAW_BRIDGE_PORT")
	if port == "" {
		port = defaultPort
	}

	// Generate OTP
	otp := generateOTP()

	// Print startup banner
	fmt.Println("========================================")
	fmt.Println("  WebClaw Bridge")
	fmt.Printf("  Version: %s\n", version)
	fmt.Println("========================================")
	fmt.Printf("\nOTP: %s\n\n", otp)
	fmt.Println("Enter this code in WebClaw to connect.")
	fmt.Printf("\nStarting server on 127.0.0.1:%s...\n", port)
	fmt.Println("(Only accessible from this machine)")
	fmt.Println()

	// Start server
	if err := StartServer(port, otp); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func printHelp() {
	fmt.Println(`WebClaw Bridge - Local companion binary for WebClaw

Usage:
  webclaw-bridge [options]

Options:
  --version, -v    Show version
  --help, -h       Show this help

Environment Variables:
  WEBCLAW_BRIDGE_PORT    Server port (default: 18800)

Security:
  - Binds only to 127.0.0.1 (localhost)
  - Requires 6-digit OTP authentication
  - Bearer tokens expire after 24 hours`)
}
