package terminals

import (
	"os/exec"
)

type GenericTerminal struct {
	BaseTerminal
	args []string
}

func NewGenericTerminal(name string, args []string) Terminal {
	return &GenericTerminal{
		BaseTerminal: BaseTerminal{Name_: name},
		args:         args,
	}
}

func (t *GenericTerminal) Open(host string) error {
	var args []string
	args = append(t.args, "ssh", host)

	cmd := exec.Command(t.Name_, args...)
	return cmd.Start()
}
