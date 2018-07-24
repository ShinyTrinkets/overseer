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

const TIME_UNIT = 100 * time.Millisecond

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
	return c
}

// Remove (un-register) a process, without stopping it.
func (ovr *Overseer) Remove(id string) {
	ovr.lock.Lock()
	defer ovr.lock.Unlock()
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
