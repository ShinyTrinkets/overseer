package overseer

import (
	"github.com/ShinyTrinkets/go-cmd"
)

const (
	DEFAULT_DELAY_START uint = 10
	DEFAULT_RETRY_TIMES uint = 3
)

type ChildProcess struct {
	cmd.Cmd
	delayStart uint // Nr of milli-seconds to delay the start
	retryTimes uint // Nr of times to restart on failure
}

// Create a new child process for the given command name and arguments.
func NewChild(name string, args ...string) *ChildProcess {
	p := cmd.NewCmdOptions(cmd.Options{false, true}, name, args...)
	return &ChildProcess{*p, DEFAULT_DELAY_START, DEFAULT_RETRY_TIMES}
}

// Sets the environment variables for the launched process.
func (p *ChildProcess) SetDir(dir string) {
	p.Lock()
	defer p.Unlock()
	p.Dir = dir
}

// Sets the working directory of the command.
func (p *ChildProcess) SetEnv(env []string) {
	p.Lock()
	defer p.Unlock()
	p.Env = env
}

// Sets the delay start in milli-seconds.
func (p *ChildProcess) SetDelayStart(delayStart uint) {
	p.Lock()
	defer p.Unlock()
	p.delayStart = delayStart
}

// Sets the times of restart in case of failure.
func (p *ChildProcess) SetRetryTimes(retryTimes uint) {
	p.Lock()
	defer p.Unlock()
	p.retryTimes = retryTimes
}
