package terminals

import (
	"fmt"
	"os/exec"
	"strings"
)

// Warp implementation
type Warp struct {
	BaseTerminal
}

func NewWarp() Terminal {
	return &Warp{
		BaseTerminal: BaseTerminal{Name_: "Warp"},
	}
}

func (t *Warp) Open(host string) error {
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
}
