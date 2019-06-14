package overseer

import (
	"testing"
)

func TestState1(t *testing.T) {
	s := INITIAL
	equals(t, s.String(), "initial")

	s = STARTING
	equals(t, s.String(), "starting")

	s = STOPPING
	equals(t, s.String(), "stopping")

	s = FINISHED
	equals(t, s.String(), "finished")
}
