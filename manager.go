package overseer

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"syscall"
	"time"

	. "github.com/ShinyTrinkets/meta-logger"
	DEATH "gopkg.in/vrecan/death.v3"
)

// Tick time unit, used when scanning the procs to see if they're alive.
const timeUnit = 100 * time.Millisecond

// Global log instance
var log Logger

// Overseer structure.
// For instantiating, it's best to use the NewOverseer() function.
type Overseer struct {
	procs    map[string]*Cmd
	lock     sync.Mutex
	stopping bool
}

// NewOverseer creates a new Overseer.
// After creating it, add the procs and call SuperviseAll.
func NewOverseer() *Overseer {
	// Setup the logs
	// TODO: This should be customizable
	log = NewLogger("overseer")
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

// ListAll returns the names of all the processes.
func (ovr *Overseer) ListAll() []string {
	ids := []string{}
	ovr.lock.Lock()
	for id := range ovr.procs {
		ids = append(ids, id)
	}
	ovr.lock.Unlock()
	sort.Strings(ids)
	return ids
}

// Add (register) a process, without starting it.
func (ovr *Overseer) Add(id string, args ...string) *Cmd {
	_, exists := ovr.procs[id]
	if exists {
		log.Info("Cannot add process, because it exists already:", Attrs{"id": id})
		return nil
	} else {
		c := NewCmdOptions(
			Options{Buffered: false, Streaming: true},
			args[0],
			args[1:]...,
		)
		log.Info("Add process:", Attrs{
			"id":   id,
			"name": c.Name,
			"args": c.Args,
		})
		ovr.lock.Lock()
		ovr.procs[id] = c
		ovr.lock.Unlock()
		return c
	}
}

// Remove (un-register) a process, if it's not running.
func (ovr *Overseer) Remove(id string) {
	stat := ovr.Status(id)
	if stat.Complete {
		log.Info("Rem process:", Attrs{"id": id})
		ovr.lock.Lock()
		delete(ovr.procs, id)
		ovr.lock.Unlock()
	} else {
		log.Info("Cannot rem process, because it's still running:", Attrs{"id": id})
	}
}

// Status returns a child process status.
// PID, Complete or not, Exit code, Error, Runtime seconds, Stdout, Stderr
func (ovr *Overseer) Status(id string) Status {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()
	return c.Status()
}

// ToJSON returns a more detailed process status, ready to be converted to JSON.
func (ovr *Overseer) ToJSON(id string) JSONProcess {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()
	return c.ToJSON()
}

// Stop the process by sending its process group a SIGTERM signal.
// The process can be started again, if needed.
func (ovr *Overseer) Stop(id string) error {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()

	if err := c.Stop(); err != nil {
		log.Error("Cannot stop process:", Attrs{"id": id})
		return err
	}

	log.Info("Stop process:", Attrs{"id": id})
	return nil
}

// Signal sends OS signal to the process group.
func (ovr *Overseer) Signal(id string, sig syscall.Signal) error {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()

	if err := c.Signal(sig); err != nil {
		log.Error("Cannot signal process:", Attrs{"id": id, "sig": sig})
		return err
	}

	log.Info("Signel process:", Attrs{"id": id, "sig": sig})
	return nil
}

// SuperviseAll is the *main* function.
// Supervise all registered processes and wait for them to finish.
func (ovr *Overseer) SuperviseAll() {
	log.Info("Start supervise all")
	for id := range ovr.procs {
		go ovr.Supervise(id)
	}
	// Check all procs every tick
	ticker := time.NewTicker(4 * timeUnit)
	for range ticker.C {
		if ovr.stopping {
			log.Info("Stop supervise all")
			break
		}

		allDone := true
		for _, p := range ovr.procs {
			stat := p.Status()
			if stat.Complete {
				continue // the process is dead, nothing to do
			}
			err := syscall.Kill(stat.PID, syscall.Signal(0))
			if err != nil {
				continue // the process is dead, nothing to do
			}
			allDone = false // if at least 1 proc is running
			break
		}

		if allDone {
			log.Info("All procs finished")
			break
		}
	}
}

// Supervise launches a process and restart it in case of failure.
func (ovr *Overseer) Supervise(id string) {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()

	delayStart := c.DelayStart
	retryTimes := c.RetryTimes
	cmdArg := fmt.Sprint(c.Name, " ", c.Args)

	// Overwrite the global log
	var log = NewLogger("proc")

	log.Info("Start overseeing process:", Attrs{"id": id})

	b := &Backoff{
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

		// Async start
		c.Start()

		// Process each line of STDOUT
		go func() {
			for line := range c.Stdout {
				log.Info(line)
				if ovr.stopping {
					break
				}
				stat := c.Status()
				if stat.Complete || stat.Exit > -1 {
					break
				}
			}
		}()

		// Process each line of STDERR
		go func() {
			for line := range c.Stderr {
				log.Error(line)
				if ovr.stopping {
					break
				}
				stat := c.Status()
				if stat.Complete || stat.Exit > -1 {
					break
				}
			}
		}()

		// Check PID from time to time, if it hasn't been killed
		// by an external signal, or from an internal error
		go func() {
			time.Sleep(2 * timeUnit)
			pid := c.Status().PID
			ticker := time.NewTicker(4 * timeUnit)
			for range ticker.C {
				err := syscall.Kill(pid, syscall.Signal(0))
				if err != nil {
					log.Info("Process has died:", Attrs{"id": id})
					break
				}
			}
		}()

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
		retryTimes--
		delayStart += uint(b.Duration())

		if retryTimes < 1 {
			log.Error("Process exited abnormally. Stopped. Error: %s", stat.Error, Attrs{"id": id})
			c.Stop() // just to make sure the status is updated
			break
		} else {
			log.Error("Process exited abnormally. Restarting [%d]. Error: %s", retryTimes+1, stat.Error, Attrs{"id": id})
		}

		ovr.lock.Lock()
		delete(ovr.procs, id)
		// overwrite the top variable
		c = c.CloneCmd()
		ovr.procs[id] = c
		ovr.lock.Unlock()
	}

	b.Reset() // clean backoff
	c.Stop()  // just to make sure the status is updated
	log.Info("Stop overseeing process:", Attrs{"id": id, "cmd": cmdArg})
}

// StopAll cycles and kills all child procs. Used when exiting the program.
func (ovr *Overseer) StopAll() {
	ovr.lock.Lock()
	ovr.stopping = true
	ovr.lock.Unlock()

	for id, c := range ovr.procs {
		c.SetRetryTimes(0) // Make sure the child doesn't restart
		ovr.Stop(id)
	}
}
