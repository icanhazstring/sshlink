package terminals

import (
	"fmt"
	"os/exec"
)

type MacOSTerminal struct {
	BaseTerminal
}

func NewMacOSTerminal() Terminal {
	return &MacOSTerminal{
		BaseTerminal: BaseTerminal{Name_: "Terminal"},
	}
}

func (t *MacOSTerminal) Open(host string) error {
	script := fmt.Sprintf(`tell application "Terminal"
	activate
	do script "ssh %s"
end tell`, host)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}
