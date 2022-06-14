//go:build windows

package overseer_test

import (
	"testing"
	"time"

	cmd "github.com/ShinyTrinkets/overseer"
	"github.com/go-test/deep"
)

func TestCmdOK(t *testing.T) {
	now := time.Now().Unix()

	p := cmd.NewCmd("echo", "foo", cmd.Options{Buffered: true})
	gotStatus := <-p.Start()
	expectStatus := cmd.Status{
		Cmd:     "echo",
		PID:     gotStatus.PID, // nondeterministic
		Exit:    0,
		Error:   nil,
		Runtime: gotStatus.Runtime, // nondeterministic
		Stdout:  []string{"foo"},
		Stderr:  []string{},
	}
	if gotStatus.StartTs < now {
		t.Error("StartTs < now")
	}
	if gotStatus.StopTs < gotStatus.StartTs {
		t.Error("StopTs < StartTs")
	}
	gotStatus.StartTs = 0
	gotStatus.StopTs = 0
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}
	if gotStatus.PID < 0 {
		t.Errorf("got PID %d, expected non-zero", gotStatus.PID)
	}
	if gotStatus.Runtime < 0 {
		t.Errorf("got runtime %f, expected non-zero", gotStatus.Runtime)
	}
}

func TestCmdClone(t *testing.T) {
	opt := cmd.Options{
		Buffered: true,
	}
	c1 := cmd.NewCmd("ls", opt)
	c1.Dir = "/tmp/"
	c1.Env = []string{"YES=please"}
	c2 := c1.Clone()

	if c1.Name != c2.Name {
		t.Errorf("got Name %s, expecting %s", c2.Name, c1.Name)
	}
	if c1.Dir != c2.Dir {
		t.Errorf("got Dir %s, expecting %s", c2.Dir, c1.Dir)
	}
	if diffs := deep.Equal(c1.Env, c2.Env); diffs != nil {
		t.Error(diffs)
	}
}

func TestCmdNonzeroExit(t *testing.T) {
	p := cmd.NewCmd("false")
	gotStatus := <-p.Start()
	expectStatus := cmd.Status{
		Cmd:     "false",
		PID:     gotStatus.PID, // nondeterministic
		Exit:    1,
		Error:   nil,
		Runtime: gotStatus.Runtime, // nondeterministic
		Stdout:  []string{},
		Stderr:  []string{},
	}
	gotStatus.StartTs = 0
	gotStatus.StopTs = 0
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}
	if gotStatus.PID < 0 {
		t.Errorf("got PID %d, expected non-zero", gotStatus.PID)
	}
	if gotStatus.Runtime < 0 {
		t.Errorf("got runtime %f, expected non-zero", gotStatus.Runtime)
	}
}

func TestCmdStop(t *testing.T) {
	p := cmd.NewCmd("sleep", "5", cmd.Options{Buffered: true})

	// Start process in bg and get chan to receive final Status when done
	statusChan := p.Start()

	// Give it a second
	time.Sleep(1 * time.Second)

	// Kill the process
	err := p.Stop()
	if err != nil {
		t.Error(err)
	}

	// The final status should be returned instantly
	timeout := time.After(1 * time.Second)
	var gotStatus cmd.Status
	select {
	case gotStatus = <-statusChan:
	case <-timeout:
		t.Fatal("timeout waiting for statusChan")
	}

	start := time.Unix(0, gotStatus.StartTs)
	stop := time.Unix(0, gotStatus.StopTs)
	d := stop.Sub(start).Seconds()
	if d < 0.90 || d > 2 {
		t.Errorf("stop - start time not between 0.9s and 2.0s: %s - %s = %f", stop, start, d)
	}
	gotStatus.StartTs = 0
	gotStatus.StopTs = 0

	expectStatus := cmd.Status{
		Cmd:     "sleep",
		PID:     gotStatus.PID, // nondeterministic
		Exit:    1,
		Error:   nil,
		Runtime: gotStatus.Runtime, // nondeterministic
		Stdout:  []string{},
		Stderr:  []string{},
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}
	if gotStatus.PID < 0 {
		t.Errorf("got PID %d, expected non-zero", gotStatus.PID)
	}
	if gotStatus.Runtime < 0 {
		t.Errorf("got runtime %f, expected non-zero", gotStatus.Runtime)
	}

	// Stop should be idempotent
	err = p.Stop()
	if err != nil {
		t.Error(err)
	}

	// Start should be idempotent, too. It just returns the same statusChan again.
	c2 := p.Start()
	if diffs := deep.Equal(statusChan, c2); diffs != nil {
		t.Error(diffs)
	}
}

func TestCmdNotStarted(t *testing.T) {
	// Call everything _but_ Start.
	p := cmd.NewCmd("echo", []string{"foo"}, cmd.Options{Buffered: true})

	gotStatus := p.Status()
	expectStatus := cmd.Status{
		Cmd:     "echo",
		PID:     0,
		Exit:    -1,
		Error:   nil,
		Runtime: 0,
		Stdout:  nil,
		Stderr:  nil,
	}
	if diffs := deep.Equal(gotStatus, expectStatus); diffs != nil {
		t.Error(diffs)
	}

	err := p.Stop()
	if err != nil {
		t.Error(err)
	}
	if p.State != cmd.INITIAL {
		t.Errorf("got state %s, expected INITIAL", p.State)
	}
}

func TestCmdNoOutput(t *testing.T) {
	// Set both output options to false to discard all output
	p := cmd.NewCmd("echo", "hell-world", cmd.Options{
		Buffered:  false,
		Streaming: false,
	})
	s := <-p.Start()
	if s.Exit != 0 {
		t.Errorf("got exit %d, expected 0", s.Exit)
	}
	if len(s.Stdout) != 0 {
		t.Errorf("got stdout, expected no output: %v", s.Stdout)
	}
	if len(s.Stderr) != 0 {
		t.Errorf("got stderr, expected no output: %v", s.Stderr)
	}
}
