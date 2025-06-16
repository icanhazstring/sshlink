package terminals

import (
	"fmt"
	"os/exec"
)

type ITerm2 struct {
	BaseTerminal
}

func NewITerm2() Terminal {
	return &ITerm2{
		BaseTerminal: BaseTerminal{Name_: "iTerm2"},
	}
}

func (t *ITerm2) Open(host string) error {
	script := fmt.Sprintf(`tell application "iTerm2"
	activate
	create window with default profile
	tell current session of current window
		write text "ssh %s"
	end tell
end tell`, host)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}
