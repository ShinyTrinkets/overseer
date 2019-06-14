// Package overseer ;
package overseer

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"syscall"
	"time"

	ml "github.com/ShinyTrinkets/meta-logger"
	DEATH "gopkg.in/vrecan/death.v3"
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
	sync.RWMutex
	procs    map[string]*Cmd
	watchers []chan *ProcessJSON
	running  bool
	stopping bool
}

// ProcessJSON public structure
type ProcessJSON struct {
	ID         string    `json:"id"`
	Group      string    `json:"group"`
	Cmd        string    `json:"cmd"`
	PID        int       `json:"PID"`
	State      string    `json:"state"`
	ExitCode   int       `json:"exitCode"` // exit code of process
	Error      error     `json:"error"`    // Go error
	StartTime  time.Time `json:"startTime"`
	Dir        string    `json:"dir"`
	DelayStart uint      `json:"delayStart"`
	RetryTimes uint      `json:"retryTimes"`
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
		procs: make(map[string]*Cmd),
	}
	// Catch death signals and stop all child procs on exit
	death := DEATH.NewDeath(syscall.SIGINT, syscall.SIGTERM)
	go death.WaitForDeathWithFunc(func() {
		ovr.StopAll()
	})
	return ovr
}

// ListAll returns the names of all the procs in alphabetic order.
func (ovr *Overseer) ListAll() []string {
	ids := []string{}
	for id := range ovr.procs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ListGroup returns the names of all the procs from a specific group.
func (ovr *Overseer) ListGroup(group string) []string {
	ids := []string{}
	for id, c := range ovr.procs {
		if c.Group == group {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

// HasProc checks if a proc has been added to the manager.
func (ovr *Overseer) HasProc(id string) bool {
	_, exists := ovr.procs[id]
	return exists
}

// Status returns a child process status
// (PID, Exit code, Error, Runtime seconds, Stdout, Stderr)
func (ovr *Overseer) Status(id string) Status {
	c := ovr.procs[id]
	return c.Status()
}

// ToJSON returns a more detailed process status, ready to be converted to JSON.
// Use this after calling ListAll() or ListGroup()
func (ovr *Overseer) ToJSON(id string) *ProcessJSON {
	ovr.Lock()
	c := ovr.procs[id]
	ovr.Unlock()
	s := c.Status()

	cmdArgs := fmt.Sprint(c.Name, " ", c.Args)
	startTime := time.Unix(0, s.StartTs)
	return &ProcessJSON{
		id,
		c.Group,
		cmdArgs,
		s.PID,
		c.State.String(),
		s.Exit,
		s.Error,
		startTime,
		c.Dir,
		c.DelayStart,
		c.RetryTimes,
	}
}

// Add (register) a process, without starting it.
func (ovr *Overseer) Add(id string, exec string, args ...interface{}) *Cmd {
	var para []string
	opts := Options{Buffered: false, Streaming: true}

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

	if exec == "" {
		log.Error("Cannot add process, no Executable defined!", Attrs{"id": id})
		return nil
	}
	_, exists := ovr.procs[id]
	if exists {
		log.Error("Cannot add process, it exists already!", Attrs{"id": id})
		return nil
	}
	c := NewCmd(exec, para, opts)
	log.Info("Add process:", Attrs{"id": id, "name": exec, "args": para})

	ovr.Lock()
	ovr.procs[id] = c
	ovr.Unlock()
	return c
}

// Remove (un-register) a process, if it's not running.
func (ovr *Overseer) Remove(id string) bool {
	_, exists := ovr.procs[id]
	if !exists {
		return false
	}
	ovr.Lock()
	defer ovr.Unlock()
	c := ovr.procs[id]

	if c.IsInitialState() || c.IsFinalState() {
		log.Info("Rem process:", Attrs{"id": id})
		delete(ovr.procs, id)
		return true
	}

	log.Info("Cannot rem process, because it's still running:", Attrs{"id": id})
	return false
}

// Stop the process by sending its process group a SIGTERM signal.
// The process can be started again, if needed.
func (ovr *Overseer) Stop(id string) error {
	c := ovr.procs[id]

	if err := c.Stop(); err != nil {
		log.Error("Cannot stop process:", Attrs{"id": id})
		return err
	}

	log.Info("Stop process:", Attrs{"id": id})
	return nil
}

// Signal sends OS signal to the process group.
func (ovr *Overseer) Signal(id string, sig syscall.Signal) error {
	c := ovr.procs[id]

	if err := c.Signal(sig); err != nil {
		log.Error("Cannot signal process:", Attrs{"id": id, "sig": sig})
		return err
	}

	log.Info("Signal process:", Attrs{"id": id, "sig": sig})
	return nil
}

// Watch allows subscribing to state changes via provided output channel.
func (ovr *Overseer) Watch(outputChan chan *ProcessJSON) {
	ovr.Lock()
	ovr.watchers = append(ovr.watchers, outputChan)
	ovr.Unlock()
}

// UnWatch allows un-subscribing from state changes.
func (ovr *Overseer) UnWatch(outputChan chan *ProcessJSON) {
	index := -1
	for i, outCh := range ovr.watchers {
		if outCh == outputChan {
			index = i
			break
		}
	}
	ovr.Lock()
	ovr.watchers = append(ovr.watchers[:index], ovr.watchers[index+1:]...)
	ovr.Unlock()
}

// SuperviseAll is the *main* function.
// Supervise all registered processes and wait for them to finish.
func (ovr *Overseer) SuperviseAll() {
	if ovr.isRunning() {
		log.Error("Supervise all is already running")
		return
	}

	ovr.setRunning(true)
	log.Info("Start supervise all")

	for id := range ovr.procs {
		go ovr.Supervise(id)
	}
	// Check all procs every tick
	// NOTE: This should probably be implemented with a WaitGroup,
	// for each Supervise() the counter goes up, and at the end it goes down
	ticker := time.NewTicker(4 * timeUnit)
	for range ticker.C {
		if ovr.isStopping() {
			log.Info("Stop supervise all")
			break
		}

		allDone := true
		for _, p := range ovr.procs {
			if p.IsFinalState() {
				continue // the process is dead, nothing to do
			}
			allDone = false // if at least 1 proc is running
			break
		}

		if allDone {
			ovr.setRunning(false)
			log.Info("All procs finished")
			break
		}
	}
}

// Supervise launches a process and restart it in case of failure.
func (ovr *Overseer) Supervise(id string) {
	ovr.Lock()
	c := ovr.procs[id]
	ovr.Unlock()

	delayStart := c.DelayStart
	retryTimes := c.RetryTimes
	cmdArg := fmt.Sprint(c.Name, " ", c.Args)

	// Overwrite the global log
	// The STDOUT and STDERR of the process
	// will also go into this log
	log := ml.NewLogger("proc")

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

	for {
		if ovr.stopping {
			break
		}
		if delayStart > 0 {
			time.Sleep(time.Duration(delayStart) * time.Millisecond)
		}

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
					s.PID,
					i.String(),
					s.Exit,
					s.Error,
					startTime,
					c.Dir,
					c.DelayStart,
					c.RetryTimes,
				}
				// Push the status change
				for _, outputChan := range ovr.watchers {
					outputChan <- info
				}
				if ovr.stopping || c.IsFinalState() {
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
				case line := <-c.Stderr:
					log.Error(line)
				default:
					if ovr.stopping || c.IsFinalState() {
						// log.Info("Close STDOUT and STDERR loop:", Attrs{"id": id})
						return
					}
					time.Sleep(2 * timeUnit)
				}
			}
		}(c)

		// Block and wait for process to finish
		stat := <-c.Start()

		// If the process didn't have any failures
		// Exited normally, no need to retry
		if stat.Exit == 0 || stat.Error == nil {
			break
		}

		if ovr.stopping {
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
			c.Stop() // just to make sure the status is updated
			break
		} else {
			log.Error("Process exited abnormally. Err: %s. Restarting [%d]. ",
				stat.Error, retryTimes+1, Attrs{"id": id, "cmd": cmdArg})
		}

		ovr.Lock()
		delete(ovr.procs, id)
		// overwrite the top variable
		c = c.Clone()
		ovr.procs[id] = c
		ovr.Unlock()

		c.Lock()
		c.DelayStart = delayStart
		c.RetryTimes = retryTimes
		c.Unlock()
	}

	c.Stop()     // just to make sure the status is updated
	bOff.Reset() // clean backoff
	log.Info("Stop overseeing process:", Attrs{"id": id, "cmd": cmdArg})
}

// StopAll cycles and kills all child procs. Used when exiting the program.
func (ovr *Overseer) StopAll() {
	ovr.setStopping(true)

	for id, c := range ovr.procs {
		c.Lock()
		c.RetryTimes = 0 // Make sure the child doesn't restart
		c.Unlock()
		ovr.Stop(id)
	}
}

func (ovr *Overseer) isRunning() bool {
	ovr.Lock()
	defer ovr.Unlock()
	return ovr.running
}

func (ovr *Overseer) setRunning(val bool) {
	ovr.Lock()
	defer ovr.Unlock()
	ovr.running = val
}

func (ovr *Overseer) isStopping() bool {
	ovr.Lock()
	defer ovr.Unlock()
	return ovr.stopping
}

func (ovr *Overseer) setStopping(val bool) {
	ovr.Lock()
	defer ovr.Unlock()
	ovr.stopping = val
}
