package terminals

import (
	"fmt"
	"os/exec"
)

type ITerm struct {
	BaseTerminal
}

func NewITerm() Terminal {
	return &ITerm{
		BaseTerminal: BaseTerminal{Name_: "iTerm"},
	}
}

func (t *ITerm) Open(host string) error {
	script := fmt.Sprintf(`tell application "iTerm"
	activate
	create window with default profile
	tell current session of current window
		write text "ssh %s"
	end tell
end tell`, host)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}
