package overseer

import (
	"fmt"
	"time"

	"github.com/ShinyTrinkets/go-cmd"
)

const (
	DEFAULT_DELAY_START uint = 25
	DEFAULT_RETRY_TIMES uint = 3
)

type ChildProcess struct {
	cmd.Cmd
	DelayStart uint // Nr of milli-seconds to delay the start
	RetryTimes uint // Nr of times to restart on failure
}

type JsonProcess struct {
	Cmd        string
	PID        int
	Complete   bool    // false if stopped or signaled
	Exit       int     // exit code of process
	Error      error   // Go error
	Runtime    float64 // seconds, zero if Cmd not started
	StartTime  time.Time
	Env        []string
	Dir        string
	DelayStart uint
	RetryTimes uint
}

// Create a new child process for the given command name and arguments.
func NewChild(name string, args ...string) *ChildProcess {
	c := cmd.NewCmdOptions(cmd.Options{false, true}, name, args...)
	return &ChildProcess{*c, DEFAULT_DELAY_START, DEFAULT_RETRY_TIMES}
}

// Clone child process.
func (o *ChildProcess) CloneChild() *ChildProcess {
	co := cmd.NewCmdOptions(cmd.Options{false, true}, o.Name, o.Args...)
	c := &ChildProcess{*co, DEFAULT_DELAY_START, DEFAULT_RETRY_TIMES}
	c.SetDir(o.Dir)
	c.SetEnv(o.Env)
	c.SetDelayStart(o.DelayStart)
	c.SetRetryTimes(o.RetryTimes)
	return c
}

func (o *ChildProcess) ToJson() JsonProcess {
	s := o.Status()
	cmd := fmt.Sprint(o.Name, " ", o.Args)
	startTime := time.Unix(0, s.StartTs)
	return JsonProcess{
		cmd,
		s.PID,
		s.Complete,
		s.Exit,
		s.Error,
		s.Runtime,
		startTime,
		o.Env,
		o.Dir,
		o.DelayStart,
		o.RetryTimes,
	}
}

// Sets the environment variables for the launched process.
func (c *ChildProcess) SetDir(dir string) {
	c.Lock()
	defer c.Unlock()
	c.Dir = dir
}

// Sets the working directory of the command.
func (c *ChildProcess) SetEnv(env []string) {
	c.Lock()
	defer c.Unlock()
	c.Env = env
}

// Sets the delay start in milli-seconds.
func (c *ChildProcess) SetDelayStart(delayStart uint) {
	c.Lock()
	defer c.Unlock()
	c.DelayStart = delayStart
}

// Sets the times of restart in case of failure.
func (c *ChildProcess) SetRetryTimes(retryTimes uint) {
	c.Lock()
	defer c.Unlock()
	c.RetryTimes = retryTimes
}
