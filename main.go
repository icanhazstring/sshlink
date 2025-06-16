package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/icanhazstring/sshlink/terminals"
)

//go:embed wrapper/sshlink_handler.m
var objcWrapperFS embed.FS

var version = "dev"

var supportedTerminals = map[string][]string{
	"terminal":  {"-e"},
	"iterm":     {},
	"kitty":     {"-e"},
	"alacritty": {"-e"},
	"wezterm":   {"start"},
	"warp":      {},
}

func main() {
	// Set up logging to a file for debugging
	setupDebugLogging()

	// Debug: Log all arguments when launched
	log.Printf("DEBUG: os.Args = %v", os.Args)
	log.Printf("DEBUG: Number of args = %d", len(os.Args))
	for i, arg := range os.Args {
		log.Printf("DEBUG: arg[%d] = %q", i, arg)
	}

	var (
		install   = flag.Bool("install", false, "Install sshlink URL handler")
		uninstall = flag.Bool("uninstall", false, "Uninstall sshlink URL handler")
		terminal  = flag.String("terminal", "terminal", "Terminal to use (terminal, iterm, warp, kitty, alacritty, wezterm)")
		showVer   = flag.Bool("version", false, "Show version")
		list      = flag.Bool("list", false, "List supported terminals")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("sshlink version %s\n", version)
		return
	}

	if *list {
		fmt.Println("Supported terminals:")
		for term := range supportedTerminals {
			fmt.Printf("  - %s\n", term)
		}
		return
	}

	if *install {
		if err := installHandler(*terminal); err != nil {
			log.Fatalf("Installation failed: %v", err)
		}
		return
	}

	if *uninstall {
		if err := uninstallHandler(); err != nil {
			log.Fatalf("Uninstallation failed: %v", err)
		}
		return
	}

	// Check if we have a URL argument
	args := flag.Args()
	log.Printf("DEBUG: flag.Args() = %v", args)
	log.Printf("DEBUG: Number of remaining args = %d", len(args))

	if len(args) == 0 {
		log.Printf("DEBUG: No URL arguments found, but app was launched")
		log.Printf("DEBUG: This might be a URL handler launch with different argument format")

		// Check if any argument looks like a URL
		for i, arg := range os.Args {
			if strings.HasPrefix(arg, "sshlink://") {
				log.Printf("DEBUG: Found sshlink URL in arg[%d]: %s", i, arg)
				urlString := arg
				terminalType := *terminal
				if *terminal == "terminal" {
					if savedTerminal := readTerminalPreference(); savedTerminal != "" {
						terminalType = savedTerminal
					}
				}

				if err := handleURL(urlString, terminalType); err != nil {
					log.Fatalf("Error: %v", err)
				}
				return
			}
		}

		log.Printf("DEBUG: No sshlink:// URL found in any argument")

		fmt.Fprintf(os.Stderr, "sshlink - SSH URL handler v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s [options] <sshlink://host>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s sshlink://192.168.1.1  # Handle SSH URL\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -install -terminal=iterm  # Install with iTerm\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -list  # Show supported terminals\n", os.Args[0])
		os.Exit(1)
	}

	urlString := args[0]
	log.Printf("DEBUG: Processing URL: %s", urlString)

	// If launched as URL handler, try to read terminal preference
	terminalType := *terminal
	if *terminal == "terminal" { // default value, might be overridden by preferences
		if savedTerminal := readTerminalPreference(); savedTerminal != "" {
			terminalType = savedTerminal
			log.Printf("DEBUG: Using saved terminal preference: %s", terminalType)
		}
	}

	log.Printf("DEBUG: About to call handleURL with URL=%s, terminal=%s", urlString, terminalType)
	if err := handleURL(urlString, terminalType); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func readTerminalPreference() string {
	if runtime.GOOS != "darwin" {
		return ""
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	prefsPath := fmt.Sprintf("%s/Library/Preferences/com.icanhazstring.sshlink.plist", homeDir)

	// Use defaults command to read preference
	cmd := exec.Command("defaults", "read", prefsPath, "defaultTerminal")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

func handleURL(urlString, terminalType string) error {
	u, err := url.Parse(urlString)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}

	if u.Scheme != "sshlink" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("no target specified")
	}

	target := strings.Replace(urlString, "sshlink://", "", 1)

	fmt.Printf("🚀 Opening SSH connection to: %s\n", target)
	return executeSSH(target, terminalType)
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

func setupDebugLogging() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	logPath := fmt.Sprintf("%s/sshlink-debug.log", homeDir)
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("=== SSHLink started ===")
}

func installHandlerMacOS(terminalName string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	appPath := fmt.Sprintf("%s/Applications/SSHLink.app", homeDir)
	contentsPath := fmt.Sprintf("%s/Contents", appPath)
	macOSPath := fmt.Sprintf("%s/MacOS", contentsPath)
	resourcesPath := fmt.Sprintf("%s/Resources", contentsPath)

	// Create directories
	if err := os.MkdirAll(macOSPath, 0755); err != nil {
		return fmt.Errorf("failed to create app bundle directories: %v", err)
	}
	if err := os.MkdirAll(resourcesPath, 0755); err != nil {
		return fmt.Errorf("failed to create resources directory: %v", err)
	}

	// Copy executable as SSHLink-real
	realExecPath := fmt.Sprintf("%s/SSHLink-real", macOSPath)
	if err := copyFile(execPath, realExecPath); err != nil {
		return fmt.Errorf("failed to copy executable: %v", err)
	}

	if err := os.Chmod(realExecPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %v", err)
	}

	// Read Objective-C wrapper source from embedded file
	objcSource, err := getObjectiveCWrapper()
	if err != nil {
		return fmt.Errorf("failed to get Objective-C wrapper source: %v", err)
	}

	// Write Objective-C source to temporary file
	objcSourcePath := fmt.Sprintf("%s/SSHLinkHandler.m", resourcesPath)
	if err := os.WriteFile(objcSourcePath, []byte(objcSource), 0644); err != nil {
		return fmt.Errorf("failed to create Objective-C source: %v", err)
	}

	// Compile the Objective-C handler
	objcBinaryPath := fmt.Sprintf("%s/SSHLink", macOSPath)
	fmt.Println("🔨 Compiling Objective-C URL handler...")

	cmd := exec.Command("clang",
		"-framework", "Foundation",
		"-framework", "AppKit",
		"-o", objcBinaryPath,
		objcSourcePath)

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to compile Objective-C handler: %v\nOutput: %s", err, output)
	}

	// Remove the source file after compilation
	os.Remove(objcSourcePath)

	// Create Info.plist
	infoPlistPath := fmt.Sprintf("%s/Info.plist", contentsPath)
	infoPlistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>SSHLink</string>
	<key>CFBundleIdentifier</key>
	<string>com.icanhazstring.sshlink</string>
	<key>CFBundleName</key>
	<string>SSHLink</string>
	<key>CFBundleDisplayName</key>
	<string>SSHLink</string>
	<key>CFBundleVersion</key>
	<string>%s</string>
	<key>CFBundleShortVersionString</key>
	<string>%s</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleURLTypes</key>
	<array>
		<dict>
			<key>CFBundleURLName</key>
			<string>SSH Link Protocol</string>
			<key>CFBundleURLSchemes</key>
			<array>
				<string>sshlink</string>
			</array>
		</dict>
	</array>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>LSUIElement</key>
	<true/>
</dict>
</plist>`, version, version)

	if err := os.WriteFile(infoPlistPath, []byte(infoPlistContent), 0644); err != nil {
		return fmt.Errorf("failed to create Info.plist: %v", err)
	}

	// Register with Launch Services
	fmt.Printf("📦 Created app bundle at: %s\n", appPath)
	fmt.Println("   ├── SSHLink (Objective-C URL handler)")
	fmt.Println("   └── SSHLink-real (Go executable)")
	fmt.Println("🔨 Compiled Objective-C Apple Event handler")
	fmt.Println("🔄 Registering with macOS Launch Services...")

	cmd = exec.Command("/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", "-f", appPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to register app with Launch Services: %v", err)
	}

	// Save terminal preference
	prefsPath := fmt.Sprintf("%s/Library/Preferences/com.icanhazstring.sshlink.plist", homeDir)
	prefsContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>defaultTerminal</key>
	<string>%s</string>
</dict>
</plist>`, terminalName)

	if err := os.WriteFile(prefsPath, []byte(prefsContent), 0644); err != nil {
		return fmt.Errorf("failed to create preferences file: %v", err)
	}

	fmt.Printf("✅ SSHLink installed successfully!\n")
	fmt.Printf("   App bundle: %s\n", appPath)
	fmt.Printf("   Default terminal: %s\n", terminalName)
	fmt.Printf("   You can now use sshlink:// URLs in your browser\n")
	fmt.Printf("   Debug logs: ~/sshlink-debug.log\n")
	fmt.Printf("\n🧪 Test installation:\n")
	fmt.Printf("   Manual test: %s sshlink://test@example.com\n", realExecPath)
	fmt.Printf("   Browser test: Click any sshlink:// URL\n")

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

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

func uninstallHandler() error {
	fmt.Printf("Uninstalling sshlink handler on %s...\n", runtime.GOOS)

	switch runtime.GOOS {
	case "darwin":
		return uninstallHandlerMacOS()
	case "linux":
		fmt.Println("ℹ️  Linux uninstallation not implemented yet.")
		return nil
	case "windows":
		fmt.Println("ℹ️  Windows uninstallation not implemented yet.")
		return nil
	default:
		return fmt.Errorf("uninstallation not supported on %s", runtime.GOOS)
	}
}

func uninstallHandlerMacOS() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	appName := "SSHLink.app"
	appPath := fmt.Sprintf("%s/Applications/%s", homeDir, appName)
	prefsPath := fmt.Sprintf("%s/Library/Preferences/com.icanhazstring.sshlink.plist", homeDir)

	// Remove app bundle
	if _, err := os.Stat(appPath); err == nil {
		if err := os.RemoveAll(appPath); err != nil {
			return fmt.Errorf("failed to remove app bundle: %v", err)
		}
		fmt.Printf("🗑️  Removed app bundle: %s\n", appPath)
	}

	// Remove preferences file
	if _, err := os.Stat(prefsPath); err == nil {
		if err := os.Remove(prefsPath); err != nil {
			return fmt.Errorf("failed to remove preferences: %v", err)
		}
		fmt.Printf("🗑️  Removed preferences: %s\n", prefsPath)
	}

	// Refresh Launch Services database
	fmt.Println("🔄 Refreshing macOS Launch Services...")
	cmd := exec.Command("/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister", "-kill", "-r", "-domain", "local", "-domain", "user")
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Warning: failed to refresh Launch Services: %v\n", err)
	}

	fmt.Println("✅ SSHLink uninstalled successfully!")
	return nil
}

func getObjectiveCWrapper() (string, error) {
	content, err := objcWrapperFS.ReadFile("wrapper/sshlink_handler.m")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded Objective-C wrapper: %v", err)
	}
	return string(content), nil
}
