package terminals

// TestTerminal interface for testing - allows external packages to inject mocks
type TestTerminal interface {
	Terminal
}

// TestCreateTerminal allows tests to override terminal creation
var TestCreateTerminal func(terminalType string) (Terminal, error)

// CreateTerminalForTest wraps the normal CreateTerminal but allows test injection
func CreateTerminalForTest(terminalType string) (Terminal, error) {
	if TestCreateTerminal != nil {
		return TestCreateTerminal(terminalType)
	}
	return CreateTerminal(terminalType)
}
