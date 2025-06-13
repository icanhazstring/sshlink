package terminals

import (
	"fmt"
	"runtime"
	"strings"
)

// CreateTerminal factory function
func CreateTerminal(terminalType string) (Terminal, error) {
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
	case "kitty":
		return NewGenericTerminal("kitty", []string{"-e"}), nil
	case "alacritty":
		return NewGenericTerminal("alacritty", []string{"-e"}), nil
	case "wezterm":
		return NewGenericTerminal("wezterm", []string{"start"}), nil
	case "terminal":
		return NewGenericTerminal("gnome-terminal", []string{"--"}), nil
	default:
		return nil, fmt.Errorf("unsupported terminal: %s", terminalType)
	}
}

func createWindowsTerminal(terminalType string) (Terminal, error) {
	// Basic Windows support
	return NewGenericTerminal("cmd", []string{"/c", "start", "cmd", "/k"}), nil
}
