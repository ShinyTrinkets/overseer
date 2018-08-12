package overseer

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestSimpleOverseer(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	ovr.Add("echo", "echo", "")
	stat := ovr.Start("echo")

	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.NotEqual(ovr.Status("echo").PID, 0, "PID shouldn't be 0")

	ovr.Add("list", "ls", "/usr/")
	stat = ovr.Start("list")

	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.NotEqual(ovr.Status("list").PID, 0, "PID shouldn't be 0")

	assert.Equal(len(ovr.ListAll()), 2, "Expected 2 procs: echo, list")

	// Should not crash
	ovr.StopAll()
}

func TestSleepOverseer(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	id := "sleep"
	p := ovr.Add(id, "sleep", "10")
	p.Start()
	time.Sleep(timeUnit)

	stat := ovr.Status(id)
	// proc is still running
	assert.Equal(stat.Exit, -1, "Exit code should be negative")
	assert.Nil(ovr.Stop(id))
	time.Sleep(timeUnit)
	// proc was killed
	assert.Equal(stat.Exit, -1, "Exit code should be negative")
}

func TestInvalidOverseer(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	id := "err1"
	ovr.Add(id, "qwertyuiop", "zxcvbnm")
	ovr.Start(id)

	time.Sleep(timeUnit)
	stat := ovr.Status(id)

	assert.Equal(stat.Exit, -1, "Exit code should be negative")
	assert.NotEqual(stat.Error, nil, "Error shouldn't be nil")

	// try to stop a dead process
	assert.Nil(ovr.Stop(id))

	id = "err2"
	ovr.Add(id, "ls", "/some_random_not_existent_path")
	ovr.Start(id)

	time.Sleep(timeUnit)
	stat = ovr.Status(id)

	assert.True(stat.Exit > 0, "Exit code should be positive")
	assert.Nil(stat.Error, "Error should be nil")
}
