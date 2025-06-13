package main

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"testing"

	"github.com/icanhazstring/sshlink/terminals"
)

// Mock terminal for testing
type MockTerminal struct {
	name      string
	openCalls []string
	shouldErr bool
}

func (m *MockTerminal) Open(host string) error {
	m.openCalls = append(m.openCalls, host)
	if m.shouldErr {
		return fmt.Errorf("mock error")
	}
	return nil
}

func (m *MockTerminal) Name() string {
	return m.name
}

func (m *MockTerminal) IsAvailable() bool {
	return true
}

func TestTerminalInterface(t *testing.T) {
	tests := []struct {
		name         string
		terminalType string
		expectError  bool
	}{
		{
			name:         "Valid terminal type",
			terminalType: "terminal",
			expectError:  false,
		},
		{
			name:         "Valid iTerm",
			terminalType: "iterm",
			expectError:  false,
		},
		{
			name:         "Valid Warp",
			terminalType: "warp",
			expectError:  false,
		},
		{
			name:         "Invalid terminal type",
			terminalType: "nonexistent",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terminal, err := terminals.CreateTerminal(tt.terminalType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if terminal == nil {
				t.Errorf("Expected terminal but got nil")
				return
			}

			// Test that we can call methods on the terminal
			name := terminal.Name()
			if name == "" {
				t.Errorf("Terminal name should not be empty")
			}

			// Test IsAvailable (may return false if terminal not installed, that's OK)
			_ = terminal.IsAvailable()
		})
	}
}

func TestMockTerminal(t *testing.T) {
	mock := &MockTerminal{name: "mock", shouldErr: false}

	// Test successful open
	err := mock.Open("test-host")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if len(mock.openCalls) != 1 || mock.openCalls[0] != "test-host" {
		t.Errorf("Expected 1 call with 'test-host', got: %v", mock.openCalls)
	}

	// Test error case
	mock.shouldErr = true
	err = mock.Open("another-host")
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestSpecificTerminals(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS terminal tests on non-macOS platform")
	}

	tests := []struct {
		name         string
		terminalType string
		constructor  func() terminals.Terminal
	}{
		{
			name:         "macOS Terminal",
			terminalType: "terminal",
			constructor:  terminals.NewMacOSTerminal,
		},
		{
			name:         "iTerm",
			terminalType: "iterm",
			constructor:  terminals.NewITerm,
		},
		{
			name:         "Warp",
			terminalType: "warp",
			constructor:  terminals.NewWarp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			terminal := tt.constructor()

			if terminal.Name() == "" {
				t.Errorf("Terminal name should not be empty")
			}

			// Test that Open doesn't panic (it will likely fail due to test environment)
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Terminal.Open() panicked: %v", r)
				}
			}()

			_ = terminal.Open("test-host")
		})
	}
}

func TestGenericTerminal(t *testing.T) {
	terminal := terminals.NewGenericTerminal("test-terminal", []string{"-e"})

	if terminal.Name() != "test-terminal" {
		t.Errorf("Expected name 'test-terminal', got '%s'", terminal.Name())
	}

	// Test that Open doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("GenericTerminal.Open() panicked: %v", r)
		}
	}()

	_ = terminal.Open("test-host")
}

func TestHandleURL(t *testing.T) {
	tests := []struct {
		name        string
		urlString   string
		terminal    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid SSH URL",
			urlString:   "sshlink://192.168.1.1",
			terminal:    "terminal",
			expectError: false,
		},
		{
			name:        "Valid SSH URL with user",
			urlString:   "sshlink://user@example.com",
			terminal:    "terminal",
			expectError: false,
		},
		{
			name:        "Valid SSH URL with port",
			urlString:   "sshlink://user@example.com:2222",
			terminal:    "terminal",
			expectError: false,
		},
		{
			name:        "Empty host",
			urlString:   "sshlink://",
			terminal:    "terminal",
			expectError: true,
			errorMsg:    "no host specified",
		},
		{
			name:        "Unsupported terminal",
			urlString:   "sshlink://192.168.1.1",
			terminal:    "nonexistent",
			expectError: true,
			errorMsg:    "unsupported terminal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the actual handleURL function from main.go
			err := handleURL(tt.urlString, tt.terminal)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestSupportedTerminals(t *testing.T) {
	expectedTerminals := []string{"terminal", "iterm", "kitty", "alacritty", "wezterm", "warp"}

	for _, terminal := range expectedTerminals {
		if _, exists := supportedTerminals[terminal]; !exists {
			t.Errorf("Expected terminal '%s' to be supported", terminal)
		}
	}

	// Test that we have the expected number of terminals
	if len(supportedTerminals) != len(expectedTerminals) {
		t.Errorf("Expected %d supported terminals, got %d", len(expectedTerminals), len(supportedTerminals))
	}
}

func TestURLParsing(t *testing.T) {
	tests := []struct {
		name         string
		urlString    string
		expectedHost string
		expectError  bool
	}{
		{
			name:         "Simple host",
			urlString:    "sshlink://192.168.1.1",
			expectedHost: "192.168.1.1",
			expectError:  false,
		},
		{
			name:         "Host with user",
			urlString:    "sshlink://user@example.com",
			expectedHost: "user@example.com",
			expectError:  false,
		},
		{
			name:         "Host with user and port",
			urlString:    "sshlink://user@example.com:2222",
			expectedHost: "user@example.com:2222",
			expectError:  false,
		},
		{
			name:         "IPv6 address",
			urlString:    "sshlink://user@[2001:db8::1]:22",
			expectedHost: "user@[2001:db8::1]:22",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the actual URL parsing logic by calling handleURL and checking it doesn't fail
			// for valid URLs (we can't easily extract the parsed host without modifying main.go)
			err := handleURL(tt.urlString, "terminal")

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none for: %s", tt.urlString)
			} else if !tt.expectError && err != nil {
				// Only fail if it's a URL parsing error, not SSH execution error
				if strings.Contains(err.Error(), "invalid URL") ||
					strings.Contains(err.Error(), "unsupported scheme") ||
					strings.Contains(err.Error(), "no host specified") {
					t.Errorf("URL parsing failed for valid URL %s: %v", tt.urlString, err)
				}
				// Ignore SSH execution errors since we can't actually SSH in tests
			}
		})
	}
}

func TestExecuteSSH(t *testing.T) {
	// Test that executeSSH can be called without panicking
	tests := []struct {
		name     string
		host     string
		terminal string
	}{
		{
			name:     "Valid host and terminal",
			host:     "192.168.1.1",
			terminal: "terminal",
		},
		{
			name:     "Valid complex host",
			host:     "user@example.com:2222",
			terminal: "terminal",
		},
		{
			name:     "Test with iterm",
			host:     "localhost",
			terminal: "iterm",
		},
		{
			name:     "Test with warp",
			host:     "localhost",
			terminal: "warp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the function can be called without panicking
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("executeSSH panicked: %v", r)
				}
			}()

			// Call the actual executeSSH function from main.go
			// It will likely fail due to missing SSH or permissions, but shouldn't panic
			_ = executeSSH(tt.host, tt.terminal)
		})
	}
}

func TestInstallHandler(t *testing.T) {
	tests := []struct {
		name     string
		terminal string
		os       string
	}{
		{
			name:     "Install on current OS",
			terminal: "terminal",
			os:       runtime.GOOS,
		},
		{
			name:     "Install with different terminal",
			terminal: "iterm",
			os:       runtime.GOOS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that installHandler doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("installHandler panicked: %v", r)
				}
			}()

			// The current implementation just prints messages, so it should always succeed
			err := installHandler(tt.terminal)
			if err != nil {
				t.Errorf("installHandler returned error: %v", err)
			}
		})
	}
}

func TestUninstallHandler(t *testing.T) {
	// Test that uninstallHandler doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("uninstallHandler panicked: %v", r)
		}
	}()

	// The current implementation just prints a message, so it should always succeed
	err := uninstallHandler()
	if err != nil {
		t.Errorf("uninstallHandler returned error: %v", err)
	}
}

// Benchmark tests
func BenchmarkHandleURL(b *testing.B) {
	urlString := "sshlink://192.168.1.1"
	terminal := "terminal"

	for i := 0; i < b.N; i++ {
		// Benchmark the actual handleURL function
		_ = handleURL(urlString, terminal)
	}
}

func BenchmarkURLParsing(b *testing.B) {
	urlString := "sshlink://user@example.com:2222"

	for i := 0; i < b.N; i++ {
		u, _ := url.Parse(urlString)
		_ = u.Host
	}
}

// Table-driven test for different URL formats
func TestVariousURLFormats(t *testing.T) {
	validURLs := []string{
		"sshlink://localhost",
		"sshlink://127.0.0.1",
		"sshlink://192.168.1.100",
		"sshlink://user@server",
		"sshlink://user@server.com",
		"sshlink://user@server:22",
		"sshlink://user@server.com:2222",
		"sshlink://root@production.example.com:22",
		"sshlink://deploy@staging-server:2200",
	}

	for _, urlString := range validURLs {
		t.Run(urlString, func(t *testing.T) {
			// Call the actual handleURL function from main.go
			err := handleURL(urlString, "terminal")

			// We expect SSH execution to fail in tests, but URL parsing should succeed
			if err != nil {
				// Only fail if it's a URL parsing/validation error, not SSH execution error
				if strings.Contains(err.Error(), "invalid URL") ||
					strings.Contains(err.Error(), "unsupported scheme") ||
					strings.Contains(err.Error(), "no host specified") {
					t.Errorf("Valid URL '%s' failed validation: %v", urlString, err)
				}
				// Ignore SSH execution errors - those are expected in test environment
			}
		})
	}
}

// Test edge cases
func TestEdgeCases(t *testing.T) {
	edgeCases := []struct {
		name        string
		urlString   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "URL with query parameters",
			urlString:   "sshlink://server?param=value",
			expectError: false, // Query params should be ignored
		},
		{
			name:        "URL with fragment",
			urlString:   "sshlink://server#fragment",
			expectError: false, // Fragment should be ignored
		},
	}

	for _, tt := range edgeCases {
		t.Run(tt.name, func(t *testing.T) {
			// Call the actual handleURL function from main.go
			err := handleURL(tt.urlString, "terminal")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none for: %s", tt.urlString)
					return
				}
				if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				// For valid URLs, ignore SSH execution errors but catch URL parsing errors
				if err != nil {
					if strings.Contains(err.Error(), "invalid URL") ||
						strings.Contains(err.Error(), "unsupported scheme") ||
						strings.Contains(err.Error(), "no host specified") {
						t.Errorf("Expected no URL parsing error but got: %v for URL: %s", err, tt.urlString)
					}
					// Ignore SSH execution errors
				}
			}
		})
	}
}
