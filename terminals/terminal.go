package terminals

import (
	"os/exec"
)

// Terminal interface defines the contract for all terminal implementations
type Terminal interface {
	Open(host string) error
	Name() string
	IsAvailable() bool
}

type BaseTerminal struct {
	Name_ string
}

func (b BaseTerminal) Name() string {
	return b.Name_
}

func (b BaseTerminal) IsAvailable() bool {
	// Check if terminal application exists
	_, err := exec.LookPath(b.Name_)
	return err == nil
}
