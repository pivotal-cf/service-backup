package executor

import "os/exec"

type ProcessStarter interface {
	Start(*exec.Cmd) error
}
