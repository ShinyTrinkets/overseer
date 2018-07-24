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

func TestCloneProcess(t *testing.T) {
	var (
		delay uint = 1
		retry uint = 9
	)

	c1 := NewChild("ls")
	c1.SetDir("/")
	c1.SetDelayStart(delay)
	c1.SetRetryTimes(retry)

	c2 := c1.CloneChild()

	if c1.Dir != c2.Dir {
		t.Fatalf("Dir was not cloned")
	}
	if c1.DelayStart != c2.DelayStart {
		t.Fatalf("Delay start was not cloned")
	}
	if c1.RetryTimes != c2.RetryTimes {
		t.Fatalf("Retry times was not cloned")
	}
}
