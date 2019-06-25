package overseer

import (
	"testing"
)

func TestState1(t *testing.T) {
	var s CmdState

	s = INITIAL
	equals(t, "initial", s.String())

	s = STARTING
	equals(t, "starting", s.String())

	s = STOPPING
	equals(t, "stopping", s.String())

	s = FINISHED
	equals(t, "finished", s.String())

	s = 99
	equals(t, "unknown", s.String())
}

func TestState2(t *testing.T) {
	var s OvrState

	s = IDLE
	equals(t, "idle", s.String())

	s = RUNNING
	equals(t, "running", s.String())

	s = STOPPING
	equals(t, "stopping", s.String())

	s = FATAL
	equals(t, "fatal", s.String())

	s = 99
	equals(t, "unknown", s.String())
}
