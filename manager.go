// Package overseer ;
package overseer

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	ml "github.com/ShinyTrinkets/meta-logger"
)

// Tick time unit, used when scanning the procs to see if they're alive.
const timeUnit = 100 * time.Millisecond

type (
	// Attrs is a type alias
	Attrs = ml.Attrs
	// Logger is a type alias
	Logger = ml.Logger
)

// Overseer structure.
// For instantiating, it's best to use the NewOverseer() function.
type Overseer struct {
	access    *sync.RWMutex
	stateLock *sync.RWMutex
	procs     sync.Map // Will contain [string]*Cmd
	watchers  []chan *ProcessJSON
	loggers   []chan *LogMsg
	state     OvrState
}

// ProcessJSON public structure
type ProcessJSON struct {
	ID         string    `json:"id"`
	Group      string    `json:"group"`
	Cmd        string    `json:"cmd"`
	Dir        string    `json:"dir"`
	PID        int       `json:"PID"`
	State      string    `json:"state"`
	ExitCode   int       `json:"exitCode"` // exit code of process
	Error      error     `json:"error"`    // Go error
	StartTime  time.Time `json:"startTime"`
	DelayStart uint      `json:"delayStart"`
	RetryTimes uint      `json:"retryTimes"`
}

type LogMsg struct {
	Type uint8
	Text string
}

// Global log instance
var log ml.Logger
var once sync.Once

// SetupLogBuilder is called to add user's provided log builder.
// It's just a wrapper around meta-logger SetupLogBuilder.
func SetupLogBuilder(b ml.LogBuilderType) {
	ml.SetupLogBuilder(b)
}

// NewOverseer creates a new process manager.
// After creating it, Add the procs and call SuperviseAll.
func NewOverseer() *Overseer {
	once.Do(func() {
		// When the logger is not defined, use the basic logger
		if ml.NewLogger == nil {
			ml.SetupLogBuilder(func(name string) Logger {
				return &ml.DefaultLogger{Name: name}
			})
		}
		// Setup the logs by calling user's provided log builder
		log = ml.NewLogger("overseer")
	})

	// Reset the random seed, for backoff jitter
	rand.Seed(time.Now().UTC().UnixNano())

	ovr := &Overseer{
		access:    &sync.RWMutex{},
		stateLock: &sync.RWMutex{},
	}

	sigChannel := make(chan os.Signal, 2)
	// Catch death signals and stop all child procs on exit
	signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		// Catch all signals and possibly react different based on signal
		for sig := range sigChannel {
			log.Error("Received signal: %v!", Attrs{"sig": sig})
			ovr.StopAll(false)
		}
	}()

	return ovr
}

// ListAll returns the names of all the procs in alphabetic order.
func (ovr *Overseer) ListAll() []string {
	ids := []string{}
	ovr.procs.Range(func(id, c interface{}) bool {
		ids = append(ids, id.(string))
		return true
	})
	sort.Strings(ids)
	return ids
}

// ListGroup returns the names of all the procs from a specific group.
func (ovr *Overseer) ListGroup(group string) []string {
	ids := []string{}
	ovr.procs.Range(func(id, c interface{}) bool {
		if c.(*Cmd).Group == group {
			ids = append(ids, id.(string))
		}
		return true
	})
	sort.Strings(ids)
	return ids
}

// HasProc checks if a proc has been added to the manager.
func (ovr *Overseer) HasProc(id string) bool {
	_, exists := ovr.procs.Load(id)
	return exists
}

// Status returns a child process status, ready to be converted to JSON.
// Use this after calling ListAll() or ListGroup()
// (PID, Exit code, Error, Runtime seconds, Stdout, Stderr)
func (ovr *Overseer) Status(id string) *ProcessJSON {
	l, ok := ovr.procs.Load(id)
	if !ok {
		return &ProcessJSON{}
	}

	c := l.(*Cmd)
	s := c.Status()

	cmdArgs := fmt.Sprint(c.Name, " ", c.Args)
	startTime := time.Unix(0, s.StartTs)
	c.stateLock.Lock()
	state := c.State.String()
	c.stateLock.Unlock()
	return &ProcessJSON{
		id,
		c.Group,
		cmdArgs,
		c.Dir,
		s.PID,
		state,
		s.Exit,
		s.Error,
		startTime,
		c.DelayStart,
		c.RetryTimes,
	}
}

// Add (register) a process, without starting it.
func (ovr *Overseer) Add(id string, exec string, args ...interface{}) *Cmd {
	var para []string
	var opts Options

	ovr.access.Lock()
	defer ovr.access.Unlock()

	if ovr.HasProc(id) {
		log.Error("Cannot add process, it exists already!", Attrs{"id": id})
		return nil
	}
	if exec == "" {
		log.Error("Cannot add process, no Executable defined!", Attrs{"id": id})
		return nil
	}

	for _, arg := range args {
		switch arg.(type) {
		case []string:
			for _, v := range arg.([]string) {
				para = append(para, fmt.Sprint(v))
			}
		case Options:
			opts = arg.(Options)
		default:
			log.Error("Unknown arg type: %T!", arg, Attrs{"id": id})
			return nil
		}
	}

	log.Info("Add process:", Attrs{"id": id, "name": exec, "args": para})
	c := NewCmd(exec, para, opts)
	ovr.procs.Store(id, c)
	return c
}

// Remove (un-register) a process, if it's not running.
func (ovr *Overseer) Remove(id string) bool {
	ovr.access.Lock()
	defer ovr.access.Unlock()

	l, ok := ovr.procs.Load(id)
	if !ok {
		return false
	}
	c := l.(*Cmd)

	if c.IsInitialState() || c.IsFinalState() {
		log.Info("Rem process:", Attrs{"id": id})
		ovr.procs.Delete(id)
		return true
	}

	log.Info("Cannot rem process, because it's still running:", Attrs{"id": id})
	return false
}

// Stop the process by sending its process group a SIGTERM signal.
// The process can be started again, if needed.
func (ovr *Overseer) Stop(id string) error {
	ovr.access.Lock()
	defer ovr.access.Unlock()

	l, ok := ovr.procs.Load(id)
	if !ok {
		log.Error("Cannot find process to stop!", Attrs{"id": id})
		return errors.New("invalid proc name")
	}
	c := l.(*Cmd)

	if err := c.Stop(); err != nil {
		log.Error("Cannot stop process:", Attrs{"id": id})
		return err
	}

	log.Info("Stop process:", Attrs{"id": id})
	return nil
}

// Signal sends OS signal to the process group.
func (ovr *Overseer) Signal(id string, sig syscall.Signal) error {
	ovr.access.Lock()
	defer ovr.access.Unlock()

	l, ok := ovr.procs.Load(id)
	if !ok {
		log.Error("Cannot find process to signal!", Attrs{"id": id})
		return errors.New("invalid proc name")
	}
	c := l.(*Cmd)

	if err := c.Signal(sig); err != nil {
		log.Error("Cannot signal process:", Attrs{"id": id, "sig": sig})
		return err
	}

	log.Info("Signal process:", Attrs{"id": id, "sig": sig})
	return nil
}

// Watch allows subscribing to state changes via provided output channel.
func (ovr *Overseer) Watch(outputChan chan *ProcessJSON) {
	ovr.access.Lock()
	defer ovr.access.Unlock()
	ovr.watchers = append(ovr.watchers, outputChan)
}

// UnWatch allows un-subscribing from state changes.
func (ovr *Overseer) UnWatch(outputChan chan *ProcessJSON) {
	index := -1
	ovr.access.Lock()
	defer ovr.access.Unlock()
	for i, outCh := range ovr.watchers {
		if outCh == outputChan {
			index = i
			break
		}
	}
	ovr.watchers = append(ovr.watchers[:index], ovr.watchers[index+1:]...)
}

// WatchLogs allows subscribing to log messages via provided output channel.
func (ovr *Overseer) WatchLogs(logChan chan *LogMsg) {
	ovr.access.Lock()
	defer ovr.access.Unlock()
	ovr.loggers = append(ovr.loggers, logChan)
}

// UnWatchLogs allows un-subscribing from log messages.
func (ovr *Overseer) UnWatchLogs(logChan chan *LogMsg) {
	index := -1
	ovr.access.Lock()
	defer ovr.access.Unlock()
	for i, outCh := range ovr.loggers {
		if outCh == logChan {
			index = i
			break
		}
	}
	ovr.loggers = append(ovr.loggers[:index], ovr.loggers[index+1:]...)
}

// SuperviseAll is the *main* function.
// Supervise all registered processes and wait for them to finish.
func (ovr *Overseer) SuperviseAll() {
	if ovr.IsRunning() {
		log.Error("SuperviseAll is already running", Attrs{"f": "SuperviseAll"})
		return
	}
	if ovr.IsStopping() {
		log.Error("Overseer is stopping", Attrs{"f": "SuperviseAll"})
		return
	}

	log.Info("Start supervise all")
	ovr.setState(RUNNING)

	var wg sync.WaitGroup

	// Launch Cmd. Lock is needed because Supervise deletes from the map of Cmds.
	ovr.procs.Range(func(id, c interface{}) bool {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			ovr.Supervise(id)
			time.Sleep(timeUnit)
		}(id.(string))
		return true
	})

	// Wait for all Cmds to complete
	wg.Wait()

	log.Info("All procs finished")
	ovr.setState(IDLE)
}

// Supervise launches a process and restart it in case of failure.
func (ovr *Overseer) Supervise(id string) int {
	ovr.access.Lock()
	l, ok := ovr.procs.Load(id)
	if !ok {
		log.Error("Cannot find process to signal!", Attrs{"id": id})
		ovr.access.Unlock()
		return -1
	}
	c := l.(*Cmd)

	if c.IsRunningState() {
		log.Error("Process is already running!", Attrs{"id": id})
		ovr.access.Unlock()
		return -1
	}
	ovr.access.Unlock()

	c.Lock()
	delayStart := c.DelayStart
	retryTimes := c.RetryTimes
	cmdArg := fmt.Sprint(c.Name, " ", c.Args)
	c.Unlock()

	// Overwrite the global log
	// The STDOUT and STDERR of the process will also go into this log
	log := ml.NewLogger("cmd:" + id)

	log.Info("Start overseeing process:", Attrs{"id": id, "cmd": cmdArg})

	jMax := delayStart
	if delayStart < 10 {
		jMax = 10
	}
	bOff := &Backoff{
		Min:    1,
		Max:    time.Duration(jMax),
		Factor: 3,
		Jitter: true,
	}

	var stat Status

	for {
		if ovr.IsStopping() {
			log.Info("Overseer is stopping", Attrs{"f": "Supervise"})
			break
		}
		if delayStart > 0 {
			time.Sleep(time.Duration(delayStart) * time.Millisecond)
		}

		// Was the retry times reset while the command was running?
		if c.getRetryTimes() == 0 && retryTimes > 0 {
			log.Error("Process is shutting down.", Attrs{"id": id, "cmd": cmdArg})
			break
		}

		// Clone the old Cmd in case of restart
		if c.IsFinalState() {
			ovr.access.Lock()
			ovr.procs.Delete(id)
			// overwrite the top variable
			c = c.Clone()
			ovr.procs.Store(id, c)
			ovr.access.Unlock()
		}

		c.Lock()
		c.DelayStart = delayStart
		c.RetryTimes = retryTimes
		c.Unlock()

		log.Info("Start process:", Attrs{
			"id":         id,
			"dir":        c.Dir,
			"cmd":        cmdArg,
			"delayStart": delayStart,
			"retryTimes": retryTimes,
		})

		// Subscribe to state changes
		go func(c *Cmd) {
			for i := range c.changeChan {
				s := c.Status()
				startTime := time.Unix(0, s.StartTs)
				info := &ProcessJSON{
					id,
					c.Group,
					cmdArg,
					c.Dir,
					s.PID,
					i.String(),
					s.Exit,
					s.Error,
					startTime,
					c.DelayStart,
					c.getRetryTimes(),
				}
				// Push the status change
				for _, outputChan := range ovr.watchers {
					outputChan <- info
				}
				if !ovr.IsRunning() || c.IsFinalState() {
					break
				}
			}
			// log.Info("Close STATE loop:", Attrs{"id": id})
		}(c)

		// Async start
		c.Start()

		// Process each line of STDOUT
		go func(c *Cmd) {
			for {
				select {
				case line := <-c.Stdout:
					log.Info(line)
					// Push the log message
					// Stdout = 1
					for _, logChan := range ovr.loggers {
						logChan <- &LogMsg{1, line}
					}
				case line := <-c.Stderr:
					log.Error(line)
					// Push the log message
					// Stderr = 2
					for _, logChan := range ovr.loggers {
						logChan <- &LogMsg{2, line}
					}
				default:
					if !ovr.IsRunning() || c.IsFinalState() {
						// log.Info("Close STDOUT and STDERR loop:", Attrs{"id": id})
						return
					}
					time.Sleep(timeUnit)
				}
			}
		}(c)

		// Block and wait for process to finish
		stat = <-c.Start()

		// If the process didn't have any failures
		// Exited normally, no need to retry
		if stat.Exit == 0 && stat.Error == nil {
			break
		}

		if ovr.IsStopping() {
			log.Info("Overseer is stopping", Attrs{"f": "Supervise"})
			break
		}

		// Decrement the number of retries and increase the start time
		if retryTimes != 0 {
			retryTimes--
		}
		delayStart += uint(bOff.Duration())

		if retryTimes < 1 {
			log.Error("Process exited abnormally. Stopped. Err: %s",
				stat.Error, Attrs{"id": id, "cmd": cmdArg})
			break
		} else {
			log.Error("Process exited abnormally. Err: %s. Restarting [%d]. ",
				stat.Error, retryTimes+1, Attrs{"id": id, "cmd": cmdArg})
		}
	}

	bOff.Reset() // clean backoff
	log.Info("Stop overseeing process:", Attrs{"id": id, "cmd": cmdArg})
	return stat.Exit
}

// StopAll cycles and kills all child procs. Used when exiting the program.
func (ovr *Overseer) StopAll(kill bool) {
	ovr.setState(STOPPING)

	ovr.procs.Range(func(id, l interface{}) bool {
		c := l.(*Cmd)
		c.Lock()
		c.RetryTimes = 0 // Make sure the child doesn't restart
		c.Unlock()
		if kill {
			ovr.Signal(id.(string), syscall.SIGKILL)
		} else {
			ovr.Stop(id.(string))
		}
		return true
	})

	ovr.access.Lock()
	time.Sleep(timeUnit)
	log.Info("All procs shutdown")
	ovr.setState(IDLE)
	ovr.access.Unlock()
}

// IsRunning returns True if SuperviseAll is running
func (ovr *Overseer) IsRunning() bool {
	ovr.stateLock.Lock()
	defer ovr.stateLock.Unlock()
	return ovr.state == RUNNING
}

// IsStopping returns True if StopAll was called and the procs
// are preparing to close
func (ovr *Overseer) IsStopping() bool {
	ovr.stateLock.Lock()
	defer ovr.stateLock.Unlock()
	return ovr.state == STOPPING
}

func (ovr *Overseer) setState(state OvrState) {
	ovr.stateLock.Lock()
	defer ovr.stateLock.Unlock()
	ovr.state = state
}
