package terminals

import (
	"fmt"
	"os/exec"
)

type LinuxTerminal struct {
	BaseTerminal
	shell string
}

func NewLinuxTerminal(name string, shell string) Terminal {
	return &LinuxTerminal{
		BaseTerminal: BaseTerminal{Name_: name},
		shell:        shell,
	}
}

func (t *LinuxTerminal) Open(host string) error {
	// For gnome-terminal: gnome-terminal --tab -- /bin/bash -c "ssh host; exec /bin/bash"
	args := []string{"--tab", "--", t.shell, "-c", fmt.Sprintf("ssh %s; exec %s", host, t.shell)}
	cmd := exec.Command(t.Name_, args...)
	return cmd.Start()
}

func (t *LinuxTerminal) IsAvailable() bool {
	// Check if the terminal executable exists
	_, err := exec.LookPath(t.Name_)
	return err == nil
}
