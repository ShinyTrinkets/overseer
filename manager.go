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

type Supervisor struct {
	procs    map[string]*ChildProcess
	stopping bool
	lock     sync.Mutex
}

func NewSupervisor() *Supervisor {
	s := &Supervisor{
		procs: make(map[string]*ChildProcess),
	}
	// Catch death signals and stop all child procs
	death := DEATH.NewDeath(syscall.SIGINT, syscall.SIGTERM)
	go death.WaitForDeathWithFunc(func() {
		s.StopAll()
	})
	return s
}
