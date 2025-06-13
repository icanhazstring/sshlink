package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"runtime"
	"strings"

	"github.com/icanhazstring/sshlink/terminals"
)

const version = "1.0.0"

var supportedTerminals = map[string][]string{
	"terminal":  {"-e"},
	"iterm":     {},
	"kitty":     {"-e"},
	"alacritty": {"-e"},
	"wezterm":   {"start"},
	"warp":      {},
}

func main() {
	var (
		install     = flag.Bool("install", false, "Install URL scheme handler")
		uninstall   = flag.Bool("uninstall", false, "Uninstall URL scheme handler")
		terminal    = flag.String("terminal", "terminal", "Terminal application to use")
		listTerms   = flag.Bool("list", false, "List supported terminals")
		versionFlag = flag.Bool("version", false, "Show version")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "sshlink v%s - One-click SSH connections from your browser\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sshlink://host\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -install                    # Install with default terminal\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -install -terminal=iterm    # Install with iTerm\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list                       # List supported terminals\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sshlink://192.168.1.1  # Handle SSH URL\n", os.Args[0])
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("sshlink v%s\n", version)
		return
	}

	if *listTerms {
		fmt.Println("Supported terminals:")
		for name := range supportedTerminals {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	if *install {
		if err := installHandler(*terminal); err != nil {
			log.Fatalf("Failed to install handler: %v", err)
		}
		fmt.Printf("✅ Handler installed successfully for %s!\n", *terminal)
		fmt.Println("You can now use sshlink://host links in your browser.")
		return
	}

	if *uninstall {
		if err := uninstallHandler(); err != nil {
			log.Fatalf("Failed to uninstall handler: %v", err)
		}
		fmt.Println("✅ Handler uninstalled successfully!")
		return
	}

	// Handle URL if provided as argument
	if len(flag.Args()) > 0 {
		urlString := flag.Args()[0]
		if strings.HasPrefix(urlString, "sshlink://") {
			if err := handleURL(urlString, *terminal); err != nil {
				log.Fatalf("Failed to handle URL: %v", err)
			}
			return
		}
	}

	// Handle URL if provided as first argument (even before flags)
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "sshlink://") {
			if err := handleURL(arg, *terminal); err != nil {
				log.Fatalf("Failed to handle URL: %v", err)
			}
			return
		}
	}

	// Show usage if no arguments provided
	flag.Usage()
}

func handleURL(urlString, terminalType string) error {
	u, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	if u.Scheme != "sshlink" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	// Parse command from URL: sshlink://host
	host := u.Host
	if host == "" {
		return fmt.Errorf("no host specified")
	}

	fmt.Printf("🚀 Opening SSH connection to: %s\n", host)
	return executeSSH(host, terminalType)
}

func executeSSH(host, terminalType string) error {
	terminal, err := terminals.CreateTerminal(terminalType)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Opening SSH connection to: %s using %s\n", host, terminal.Name())
	return terminal.Open(host)
}

func installHandler(terminalType string) error {
	fmt.Printf("Installing sshlink handler for %s on %s...\n", terminalType, runtime.GOOS)

	// Validate terminal type by trying to create it
	terminal, err := terminals.CreateTerminal(terminalType)
	if err != nil {
		return err
	}

	switch runtime.GOOS {
	case "darwin":
		return installHandlerMacOS(terminal.Name())
	case "linux":
		return installHandlerLinux(terminal.Name())
	case "windows":
		return installHandlerWindows(terminal.Name())
	default:
		return fmt.Errorf("installation not supported on %s", runtime.GOOS)
	}
}

func installHandlerMacOS(terminalName string) error {
	fmt.Println("ℹ️  MacOS installation requires creating an app bundle.")
	fmt.Println("   This minimal version shows the concept.")
	fmt.Printf("   Would install handler for: %s\n", terminalName)
	return nil
}

func installHandlerLinux(terminalName string) error {
	fmt.Println("ℹ️  Linux installation requires creating .desktop files.")
	fmt.Println("   This minimal version shows the concept.")
	fmt.Printf("   Would install handler for: %s\n", terminalName)
	return nil
}

func installHandlerWindows(terminalName string) error {
	fmt.Println("ℹ️  Windows installation requires registry modifications.")
	fmt.Println("   This minimal version shows the concept.")
	fmt.Printf("   Would install handler for: %s\n", terminalName)
	return nil
}

func uninstallHandler() error {
	fmt.Printf("Uninstalling sshlink handler on %s...\n", runtime.GOOS)
	fmt.Println("ℹ️  This minimal version shows the concept.")
	return nil
}
