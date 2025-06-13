package main

import (
	"strings"
	"testing"

	"github.com/icanhazstring/sshlink/terminals"
)

// MockTerminal captures the commands passed to it
type MockTerminal struct {
	capturedCommand string
}

func (m *MockTerminal) Open(host string) error {
	m.capturedCommand = host
	return nil
}

func (m *MockTerminal) Name() string {
	return "MockTerminal"
}

func (m *MockTerminal) IsAvailable() bool {
	return true
}

func TestSSHLinkExecution(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedHost string
	}{
		{
			name:         "Simple host",
			url:          "sshlink://example.com",
			expectedHost: "example.com",
		},
		{
			name:         "User and host",
			url:          "sshlink://user@example.com",
			expectedHost: "user@example.com",
		},
		{
			name:         "User, host and port",
			url:          "sshlink://user@example.com:2222",
			expectedHost: "user@example.com:2222",
		},
		{
			name:         "IPv6 address",
			url:          "sshlink://user@[2001:db8::1]:22",
			expectedHost: "user@[2001:db8::1]:22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock terminal for this test
			mock := &MockTerminal{}

			// Override terminal creation with our mock
			originalTestCreateTerminal := terminals.TestCreateTerminal
			defer func() { terminals.TestCreateTerminal = originalTestCreateTerminal }()

			terminals.TestCreateTerminal = func(terminalType string) (terminals.Terminal, error) {
				return mock, nil
			}

			// Execute the sshlink workflow: ./sshlink sshlink://user@example.com
			err := handleURL(tt.url, "terminal")
			if err != nil {
				t.Fatalf("handleURL failed: %v", err)
			}

			// Verify the mock captured the correct SSH command
			if mock.capturedCommand != tt.expectedHost {
				t.Errorf("Expected SSH command '%s', but terminal.Open() received '%s'",
					tt.expectedHost, mock.capturedCommand)
			}

			t.Logf("✅ sshlink correctly passed '%s' to terminal.Open()", mock.capturedCommand)
		})
	}
}

func TestSSHLinkExecutionErrors(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty target",
			url:         "sshlink://",
			expectError: true,
			errorMsg:    "no target specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock terminal
			mock := &MockTerminal{}

			// Override terminal creation with our mock
			originalTestCreateTerminal := terminals.TestCreateTerminal
			defer func() { terminals.TestCreateTerminal = originalTestCreateTerminal }()

			terminals.TestCreateTerminal = func(terminalType string) (terminals.Terminal, error) {
				return mock, nil
			}

			// Execute handleURL and expect error
			err := handleURL(tt.url, "terminal")

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
				t.Logf("✅ sshlink correctly rejected invalid URL: %v", err)
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
