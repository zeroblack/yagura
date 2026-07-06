package execx

import (
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Harden makes a command safe to kill: the cancel signal reaches the whole
// process group (a killed git or gh can leave helpers behind that inherit the
// output pipe and would otherwise survive), and WaitDelay force-closes the
// pipes so Wait can never block on a descriptor a surviving process still
// holds after the command itself is gone.
func Harden(cmd *exec.Cmd, waitDelay time.Duration) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if err == syscall.ESRCH {
			return os.ErrProcessDone
		}
		return err
	}
	cmd.WaitDelay = waitDelay
}
