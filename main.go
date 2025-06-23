package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"

	"github.com/icanhazstring/sshlink/terminals"
)

//go:embed wrapper/sshlink_handler.m
var objcWrapperFS embed.FS

var version = "dev"

var supportedDarwinTerminals = map[string][]string{
	"terminal": {"-e"},
	"iterm":    {},
	"iterm2":   {},
	"warp":     {},
}

var supportedLinuxTerminals = map[string][]string{
	"gnome-terminal": {"--tab", "--"},
}

var supportedWindowsTerminals = map[string][]string{}

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
		switch runtime.GOOS {
		case "darwin":
			for term := range supportedDarwinTerminals {
				fmt.Printf("  - %s\n", term)
			}
		case "linux":
			for term := range supportedLinuxTerminals {
				fmt.Printf("  - %s\n", term)
			}
		case "windows":
			for term := range supportedWindowsTerminals {
				fmt.Printf("  - %s\n", term)
			}
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
	switch runtime.GOOS {
	case "darwin":
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

	case "linux":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}

		prefsFile := filepath.Join(homeDir, ".config", "sshlink", "config")
		content, err := os.ReadFile(prefsFile)
		if err != nil {
			return ""
		}

		// Parse simple key=value format
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "terminal=") {
				return strings.TrimPrefix(line, "terminal=")
			}
		}

	default:
		return ""
	}

	return ""
}

func readShellPreference() string {
	if runtime.GOOS != "linux" {
		return "/bin/bash" // fallback for non-Linux
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/bin/bash"
	}

	prefsFile := filepath.Join(homeDir, ".config", "sshlink", "config")
	content, err := os.ReadFile(prefsFile)
	if err != nil {
		return detectUserShell() // fallback to detection
	}

	// Parse simple key=value format
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "shell=") {
			return strings.TrimPrefix(line, "shell=")
		}
	}

	return detectUserShell() // fallback to detection
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
	// For Linux, set the user shell in the factory before creating terminal
	if runtime.GOOS == "linux" {
		userShell := readShellPreference()
		terminals.SetUserShell(userShell)
		log.Printf("DEBUG: Set user shell to: %s", userShell)
	}

	terminal, err := terminals.CreateTerminal(terminalType)
	if err != nil {
		return err
	}

	fmt.Printf("🚀 Opening SSH connection to: %s using %s\n", host, terminal.Name())
	return terminal.Open(host)
}

func installHandler(terminalType string) error {
	fmt.Printf("Installing sshlink handler for %s on %s...\n", terminalType, runtime.GOOS)

	// For Linux, set the user shell in the factory before creating terminal
	if runtime.GOOS == "linux" {
		userShell := readShellPreference()
		terminals.SetUserShell(userShell)
		log.Printf("DEBUG: Set user shell to: %s", userShell)
	}

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
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	// Get the absolute path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Detect user's shell
	userShell := detectUserShell()
	fmt.Printf("🐚 Detected shell: %s\n", userShell)

	// Create directories if they don't exist
	appDir := filepath.Join(usr.HomeDir, ".local", "share", "applications")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create applications directory: %v", err)
	}

	fmt.Printf("📦 Installing sshlink handler for Linux...\n")
	fmt.Printf("   Terminal: %s\n", terminalName)
	fmt.Printf("   Executable: %s\n", execPath)

	// Create the desktop file
	desktopFile := filepath.Join(appDir, "sshlink.desktop")
	if err := createDesktopFile(desktopFile, execPath, terminalName); err != nil {
		return fmt.Errorf("failed to create desktop file: %v", err)
	}

	fmt.Printf("📄 Created desktop file: %s\n", desktopFile)

	// Save terminal preference and shell
	prefsDir := filepath.Join(usr.HomeDir, ".config", "sshlink")
	if err := os.MkdirAll(prefsDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	prefsFile := filepath.Join(prefsDir, "config")
	prefsContent := fmt.Sprintf("terminal=%s\nshell=%s\n", terminalName, userShell)
	if err := os.WriteFile(prefsFile, []byte(prefsContent), 0644); err != nil {
		return fmt.Errorf("failed to save terminal preference: %v", err)
	}

	fmt.Printf("⚙️  Saved preferences: %s\n", prefsFile)
	fmt.Printf("   Terminal: %s\n", terminalName)
	fmt.Printf("   Shell: %s\n", userShell)

	// Register the protocol handler
	if err := registerProtocolHandler(); err != nil {
		return fmt.Errorf("failed to register protocol handler: %v", err)
	}

	fmt.Printf("✅ SSHLink installed successfully for Linux!\n")
	fmt.Printf("   Desktop file: %s\n", desktopFile)
	fmt.Printf("   Default terminal: %s\n", terminalName)
	fmt.Printf("   Default shell: %s\n", userShell)
	fmt.Printf("   Config: %s\n", prefsFile)
	fmt.Printf("   You can now use sshlink:// URLs in your browser\n")
	fmt.Printf("   Debug logs: ~/sshlink-debug.log\n")
	fmt.Printf("\n🧪 Test installation:\n")
	fmt.Printf("   Manual test: %s sshlink://test@example.com\n", execPath)
	fmt.Printf("   Browser test: Click any sshlink:// URL\n")

	return nil
}

func registerProtocolHandler() error {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		usr, err := user.Current()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %v", err)
		}
		homeDir = usr.HomeDir
	}

	appDir := filepath.Join(homeDir, ".local", "share", "applications")

	fmt.Println("🔄 Updating desktop database...")
	// Update desktop database
	updateDatabaseCommand := exec.Command("update-desktop-database", appDir)
	if err := updateDatabaseCommand.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not update desktop database: %v\n", err)
		// Don't return error as this is not critical
	}

	fmt.Println("🔗 Registering protocol handler...")
	// Register with xdg-mime
	registerCommand := exec.Command("xdg-mime", "default", "sshlink.desktop", "x-scheme-handler/sshlink")
	if err := registerCommand.Run(); err != nil {
		return fmt.Errorf("failed to register with xdg-mime: %v (make sure xdg-utils is installed)", err)
	}

	// Verify registration
	fmt.Println("✓ Verifying protocol registration...")
	verifyCommand := exec.Command("xdg-mime", "query", "default", "x-scheme-handler/sshlink")
	if output, err := verifyCommand.Output(); err == nil {
		result := strings.TrimSpace(string(output))
		if result == "sshlink.desktop" {
			fmt.Println("✓ Protocol handler registered successfully")
		} else {
			fmt.Printf("⚠️  Warning: Expected 'sshlink.desktop', got '%s'\n", result)
		}
	}

	return nil
}

func createDesktopFile(filePath, execPath, terminalName string) error {
	desktopTemplate := `[Desktop Entry]
Type=Application
Name=SSH Link Handler
Comment=Handle sshlink:// URLs
Exec={{.ExecPath}} -terminal={{.Terminal}} %u
Icon=utilities-terminal
StartupNotify=false
NoDisplay=true
MimeType=x-scheme-handler/sshlink;
Categories=Network;
Terminal=false
`

	tmpl, err := template.New("desktop").Parse(desktopTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse desktop template: %v", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create desktop file: %v", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, struct {
		ExecPath string
		Terminal string
	}{
		ExecPath: execPath,
		Terminal: terminalName,
	}); err != nil {
		return fmt.Errorf("failed to write desktop file: %v", err)
	}

	// Make the desktop file executable
	if err := os.Chmod(filePath, 0755); err != nil {
		return fmt.Errorf("failed to make desktop file executable: %v", err)
	}

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
		return uninstallHandlerLinux()
	case "windows":
		fmt.Println("ℹ️  Windows uninstallation not implemented yet.")
		return nil
	default:
		return fmt.Errorf("uninstallation not supported on %s", runtime.GOOS)
	}
}

func uninstallHandlerLinux() error {
	usr, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to get current user: %v", err)
	}

	fmt.Println("🗑️  Uninstalling sshlink handler for Linux...")

	// Remove desktop file
	desktopFile := filepath.Join(usr.HomeDir, ".local", "share", "applications", "sshlink.desktop")
	if _, err := os.Stat(desktopFile); err == nil {
		if err := os.Remove(desktopFile); err != nil {
			return fmt.Errorf("failed to remove desktop file: %v", err)
		}
		fmt.Printf("🗑️  Removed desktop file: %s\n", desktopFile)
	}

	// Remove config directory
	configDir := filepath.Join(usr.HomeDir, ".config", "sshlink")
	if _, err := os.Stat(configDir); err == nil {
		if err := os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("failed to remove config directory: %v", err)
		}
		fmt.Printf("🗑️  Removed config: %s\n", configDir)
	}

	// Update desktop database
	appDir := filepath.Join(usr.HomeDir, ".local", "share", "applications")
	fmt.Println("🔄 Updating desktop database...")
	updateDatabaseCommand := exec.Command("update-desktop-database", appDir)
	if err := updateDatabaseCommand.Run(); err != nil {
		fmt.Printf("⚠️  Warning: Could not update desktop database: %v\n", err)
	}

	// Try to unregister the protocol handler
	fmt.Println("🔗 Attempting to unregister protocol handler...")
	// Note: There's no direct way to "unregister" with xdg-mime,
	// but removing the desktop file and updating the database should be sufficient

	fmt.Println("✅ SSHLink uninstalled successfully from Linux!")
	fmt.Println("   Note: You may need to restart your browser for changes to take effect")

	return nil
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

func detectUserShell() string {
	// First, try to get shell from environment
	if shell := os.Getenv("SHELL"); shell != "" {
		return shell
	}

	// Fallback: try to get from /etc/passwd
	if usr, err := user.Current(); err == nil {
		cmd := exec.Command("getent", "passwd", usr.Username)
		if output, err := cmd.Output(); err == nil {
			// Parse /etc/passwd format: username:x:uid:gid:comment:home:shell
			fields := strings.Split(strings.TrimSpace(string(output)), ":")
			if len(fields) >= 7 && fields[6] != "" {
				return fields[6]
			}
		}
	}

	// Final fallback
	return "/bin/bash"
}
