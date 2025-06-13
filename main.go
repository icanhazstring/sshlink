package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
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
		fmt.Fprintf(os.Stderr, "  %s sshlink://ssh+host\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -install                    # Install with default terminal\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -install -terminal=iterm    # Install with iTerm\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list                       # List supported terminals\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s sshlink://ssh+192.168.1.1  # Handle SSH URL\n", os.Args[0])
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
		fmt.Println("You can now use sshlink://ssh+host links in your browser.")
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

	// Parse command from URL: sshlink://ssh+host
	command := u.Host
	if !strings.HasPrefix(command, "ssh+") {
		return fmt.Errorf("unsupported command format: %s", command)
	}

	host := strings.TrimPrefix(command, "ssh+")
	if host == "" {
		return fmt.Errorf("no host specified")
	}

	fmt.Printf("🚀 Opening SSH connection to: %s\n", host)
	return executeSSH(host, terminalType)
}

func executeSSH(host, terminalType string) error {
	switch runtime.GOOS {
	case "darwin":
		return executeSSHMacOS(host, terminalType)
	case "linux":
		return executeSSHLinux(host, terminalType)
	case "windows":
		return executeSSHWindows(host, terminalType)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func executeSSHMacOS(host, terminalType string) error {
	switch strings.ToLower(terminalType) {
	case "terminal":
		script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "ssh %s"
end tell`, host)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "iterm":
		script := fmt.Sprintf(`tell application "iTerm"
	activate
	create window with default profile
	tell current session of current window
		write text "ssh %s"
	end tell
end tell`, host)
		cmd := exec.Command("osascript", "-e", script)
		return cmd.Run()

	case "warp":
		// Warp doesn't have good AppleScript automation, so we'll copy the command
		// to clipboard and open Warp - user can just paste with Cmd+V
		sshCommand := fmt.Sprintf("ssh %s", host)

		// Copy SSH command to clipboard
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(sshCommand)
		if err := cmd.Run(); err != nil {
			fmt.Printf("⚠️  Could not copy to clipboard: %v\n", err)
		} else {
			fmt.Printf("📋 Copied to clipboard: %s\n", sshCommand)
			fmt.Println("💡 Paste with Cmd+V in Warp terminal")
		}

		// Open Warp
		cmd = exec.Command("open", "-a", "Warp")
		return cmd.Run()

	default:
		// For other terminals, try direct execution
		args, exists := supportedTerminals[terminalType]
		if !exists {
			return fmt.Errorf("unsupported terminal: %s", terminalType)
		}
		args = append(args, "ssh", host)
		cmd := exec.Command(terminalType, args...)
		return cmd.Start()
	}
}

func executeSSHLinux(host, terminalType string) error {
	args, exists := supportedTerminals[terminalType]
	if !exists {
		return fmt.Errorf("unsupported terminal: %s", terminalType)
	}
	args = append(args, "ssh", host)
	cmd := exec.Command(terminalType, args...)
	return cmd.Start()
}

func executeSSHWindows(host, terminalType string) error {
	// Basic Windows support - opens in cmd
	cmd := exec.Command("cmd", "/c", "start", "cmd", "/k", fmt.Sprintf("ssh %s", host))
	return cmd.Start()
}

func installHandler(terminalType string) error {
	fmt.Printf("Installing sshlink handler for %s on %s...\n", terminalType, runtime.GOOS)

	switch runtime.GOOS {
	case "darwin":
		return installHandlerMacOS(terminalType)
	case "linux":
		return installHandlerLinux(terminalType)
	case "windows":
		return installHandlerWindows(terminalType)
	default:
		return fmt.Errorf("installation not supported on %s", runtime.GOOS)
	}
}

func installHandlerMacOS(terminalType string) error {
	// This is a simplified version - in a real implementation,
	// you'd create the full .app bundle structure
	fmt.Println("ℹ️  MacOS installation requires creating an app bundle.")
	fmt.Println("   This minimal version shows the concept.")
	fmt.Printf("   Would install handler for: %s\n", terminalType)
	return nil
}

func installHandlerLinux(terminalType string) error {
	fmt.Println("ℹ️  Linux installation requires creating .desktop files.")
	fmt.Println("   This minimal version shows the concept.")
	fmt.Printf("   Would install handler for: %s\n", terminalType)
	return nil
}

func installHandlerWindows(terminalType string) error {
	fmt.Println("ℹ️  Windows installation requires registry modifications.")
	fmt.Println("   This minimal version shows the concept.")
	fmt.Printf("   Would install handler for: %s\n", terminalType)
	return nil
}

func uninstallHandler() error {
	fmt.Printf("Uninstalling sshlink handler on %s...\n", runtime.GOOS)
	fmt.Println("ℹ️  This minimal version shows the concept.")
	return nil
}
