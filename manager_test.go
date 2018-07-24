package overseer

import (
	"testing"
	"time"
)

func TestSimpleOverseer(t *testing.T) {
	ovr := NewOverseer()

	ovr.Add("echo", "echo", "")
	stat := ovr.Start("echo")

	if stat.Exit != 0 {
		t.Fatalf("Exit code should be 0")
	}
	if ovr.GetPID("echo") == 0 {
		t.Fatalf("PID shouldn't be 0")
	}

	ovr.Add("list", "ls", "/usr/")
	stat = ovr.Start("list")

	if stat.Exit != 0 {
		t.Fatalf("Exit code should be 0")
	}
	if ovr.GetPID("list") == 0 {
		t.Fatalf("PID shouldn't be 0")
	}

	if len(ovr.ListAll()) != 2 {
		t.Fatalf("Expected 2 procs: echo, list")
	}

	// Should not crash
	ovr.StopAll()
}

func TestSleepOverseer(t *testing.T) {
	id := "sleep"
	ovr := NewOverseer()

	p := ovr.Add(id, "sleep", "10")
	p.Start()
	time.Sleep(TIME_UNIT)

	stat := ovr.Status(id)
	if stat.Exit != -1 {
		t.Fatalf("Exit code should be negative")
	}

	err := ovr.Stop(id)
	if err != nil {
		t.Fatalf("Process should stop successfully")
	}
}
