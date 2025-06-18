package terminals

import (
	"fmt"
	"runtime"
	"strings"
)

// TestCreateTerminal allows tests to override terminal creation
var TestCreateTerminal func(terminalType string) (Terminal, error)

// CreateTerminal factory function
func CreateTerminal(terminalType string) (Terminal, error) {
	if TestCreateTerminal != nil {
		return TestCreateTerminal(terminalType)
	}

	switch runtime.GOOS {
	case "darwin":
		return createMacOSTerminal(terminalType)
	case "linux":
		return createLinuxTerminal(terminalType)
	case "windows":
		return createWindowsTerminal(terminalType)
	default:
		return nil, fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func createMacOSTerminal(terminalType string) (Terminal, error) {
	switch strings.ToLower(terminalType) {
	case "terminal":
		return NewMacOSTerminal(), nil
	case "iterm":
		return NewITerm(), nil
	case "iterm2":
		return NewITerm2(), nil
	case "warp":
		return NewWarp(), nil
	case "kitty":
		return NewGenericTerminal("kitty", []string{"-e"}), nil
	case "alacritty":
		return NewGenericTerminal("alacritty", []string{"-e"}), nil
	case "wezterm":
		return NewGenericTerminal("wezterm", []string{"start"}), nil
	default:
		return nil, fmt.Errorf("unsupported terminal: %s", terminalType)
	}
}

func createLinuxTerminal(terminalType string) (Terminal, error) {
	switch strings.ToLower(terminalType) {
	case "gnome-terminal":
		return NewGenericTerminal("gnome-terminal", []string{"--tab", "--", "/bin/bash", "-c"}), nil
	default:
		return nil, fmt.Errorf("unsupported terminal: %s", terminalType)
	}
}

func createWindowsTerminal(terminalType string) (Terminal, error) {
	// Basic Windows support
	return NewGenericTerminal("cmd", []string{"/c", "start", "cmd", "/k"}), nil
}
