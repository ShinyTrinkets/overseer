package overseer

import (
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/ShinyTrinkets/go-cmd"
	"github.com/rs/zerolog/log"
	DEATH "gopkg.in/vrecan/death.v3"
)

// Tick time unit, used when scanning the procs to see if they're alive.
const TIME_UNIT = 100 * time.Millisecond

// The Overseer structure.
// For instantiating, it's best to use the NewOverseer() function.
type Overseer struct {
	procs    map[string]*ChildProcess
	lock     sync.Mutex
	stopping bool
}

func NewOverseer() *Overseer {
	ovr := &Overseer{
		procs: make(map[string]*ChildProcess),
	}
	// Catch death signals and stop all child procs
	death := DEATH.NewDeath(syscall.SIGINT, syscall.SIGTERM)
	go death.WaitForDeathWithFunc(func() {
		ovr.StopAll()
	})
	return ovr
}

// Return the names of all the processes.
func (ovr *Overseer) ListAll() []string {
	ids := []string{}
	for id := range ovr.procs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Add (register) a process, without starting it.
func (ovr *Overseer) Add(id string, args ...string) *ChildProcess {
	ovr.lock.Lock()
	defer ovr.lock.Unlock()

	c := NewChild(args[0], args[1:]...)
	ovr.procs[id] = c
	log.Debug().Str("Proc", id).Msgf("Add process: %s %v", c.Name, c.Args)
	return c
}

// Remove (un-register) a process, without stopping it.
func (ovr *Overseer) Remove(id string) {
	ovr.lock.Lock()
	defer ovr.lock.Unlock()
	log.Debug().Str("Proc", id).Msg("Rem process")
	delete(ovr.procs, id)
}

// Simply start the process and block until it finishes.
// The process can be started again, if needed.
func (ovr *Overseer) Start(id string) cmd.Status {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()

	log.Info().Str("Proc", id).Msg("Start process")
	return <-c.Start()
}

// Stop the process by sending its process group a SIGTERM signal.
// The process can be started again, if needed.
func (ovr *Overseer) Stop(id string) error {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()

	err := c.Stop()
	if err != nil {
		log.Error().Str("Proc", id).Msg("Cannot stop process")
		return err
	}

	log.Info().Str("Proc", id).Msg("Stop process")
	return nil
}

// This is the *main* function.
// Supervise all registered processes and wait for them to finish.
func (ovr *Overseer) SuperviseAll() {
	log.Info().Msg("Start supervise all")
	for id := range ovr.procs {
		go ovr.Supervise(id)
	}
	// Check all procs every tick
	ticker := time.NewTicker(4 * TIME_UNIT)
	for range ticker.C {
		if ovr.stopping {
			log.Info().Msg("Stop supervise all")
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
			log.Info().Msg("All procs finished")
			break
		}
	}
}

// Start a process and restart it in case of failure.
func (ovr *Overseer) Supervise(id string) {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()

	delayStart := c.DelayStart
	retryTimes := c.RetryTimes

	log.Info().
		Str("Proc", id).Str("Dir", c.Dir).
		Uint("DelayStart", delayStart).
		Uint("RetryTimes", retryTimes).
		Msg("Start overseeing process")

	for {
		if ovr.stopping {
			break
		}
		if delayStart > 0 {
			time.Sleep(time.Duration(delayStart) * time.Millisecond)
		}

		log.Debug().Str("Proc", id).Msgf("Start process: %s %v", c.Name, c.Args)

		// Async start
		c.Start()

		// Process each line of STDOUT
		go func() {
			for line := range c.Stdout {
				if ovr.stopping {
					break
				}
				stat := c.Status()
				if stat.Complete || stat.Exit > -1 {
					break
				}
				log.Debug().Msg(line)
			}
		}()

		// Process each line of STDERR
		go func() {
			for line := range c.Stderr {
				if ovr.stopping {
					break
				}
				stat := c.Status()
				if stat.Complete || stat.Exit > -1 {
					break
				}
				log.Debug().Msg(line)
			}
		}()

		// Check PID from time to time, if it hasn't been killed
		// by an external signal, or from an internal error
		go func() {
			time.Sleep(2 * TIME_UNIT)
			pid := c.Status().PID
			ticker := time.NewTicker(4 * TIME_UNIT)
			for range ticker.C {
				err := syscall.Kill(pid, syscall.Signal(0))
				if err != nil {
					log.Debug().Str("Proc", id).Err(err).Msg("Process has died")
					break
				}
			}
		}()

		// Block and wait for process to finish
		stat := <-c.Start()

		// If the process didn't have any failures
		// Exit normally, no need to retry
		if stat.Exit == 0 || stat.Error == nil {
			break
		}

		if ovr.stopping {
			break
		}

		retryTimes--

		if retryTimes < 1 {
			log.Warn().Str("Proc", id).Err(stat.Error).Msgf("Process exited abnormally. Stopped.")
			c.Stop() // just to make sure the status is updated
			break
		} else {
			log.Warn().Str("Proc", id).Err(stat.Error).
				Msgf("Process exited abnormally. Restarting [%d]", retryTimes+1)
		}

		ovr.lock.Lock()
		delete(ovr.procs, id)
		c = c.CloneChild()
		ovr.procs[id] = c
		ovr.lock.Unlock()
	}

	c.Stop() // just to make sure the status is updated
	log.Info().Str("Proc", id).Msg("Stop overseeing process")
}

// Get a child process status.
// PID, Complete or not, Exit code, Error, Runtime seconds, Stdout, Stderr
func (ovr *Overseer) Status(id string) cmd.Status {
	ovr.lock.Lock()
	c := ovr.procs[id]
	ovr.lock.Unlock()
	return c.Status()
}

// Get the PID of the child process.
func (ovr *Overseer) GetPID(id string) int {
	stat := ovr.Status(id)
	return stat.PID
}

// Send OS signal to a child process.
func (ovr *Overseer) Signal(id string, sig syscall.Signal) error {
	pid := ovr.GetPID(id)
	return syscall.Kill(-pid, sig)
}

// Cycle and kill all child procs. Used when exiting the program.
func (ovr *Overseer) StopAll() {
	ovr.lock.Lock()
	ovr.stopping = true
	ovr.lock.Unlock()

	for id, c := range ovr.procs {
		c.SetRetryTimes(0) // Make sure the child doesn't restart
		ovr.Stop(id)
	}
}
