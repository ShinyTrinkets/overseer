package overseer

// Command states
type CmdState uint

const (
	INITIAL   CmdState = iota
	STARTING           = 10
	RUNNING            = 20
	STOPPING           = 30
	INTERRUPT          = 40 // final state (used when stopped or signaled)
	FINISHED           = 50 // final state (used then was a natural exit)
	FATAL              = 60 // final state (used when was an error while starting)
	UNKNOWN            = 99
)

func (p CmdState) String() string {
	switch p {
	case INITIAL:
		return "initial"
	case STARTING:
		return "starting"
	case RUNNING:
		return "running"
	case STOPPING:
		return "stopping"
	case INTERRUPT:
		return "interrupted"
	case FINISHED:
		return "finished"
	case FATAL:
		return "fatal"
	default:
		return "unknown"
	}
}
