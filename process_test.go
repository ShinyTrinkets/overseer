package overseer

import (
	"testing"
)

func TestSimpleProcess(t *testing.T) {
	c := NewChild("ls")

	if c.DelayStart != DEFAULT_DELAY_START {
		t.Fatalf("Default delay start is wrong")
	}
	if c.RetryTimes != DEFAULT_RETRY_TIMES {
		t.Fatalf("Default retry times is wrong")
	}

	c.SetDir("/")
	if c.Dir != "/" {
		t.Fatalf("Could not set dir")
	}
}
