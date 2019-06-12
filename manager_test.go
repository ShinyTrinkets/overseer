// Package overseer ;
package overseer

// Currently using testify/assert here
// and go-test/deep for cmd_test
// Not optimal
import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	logr "github.com/ShinyTrinkets/meta-logger"
	"github.com/azer/logger"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logr.SetupLogBuilder(func(name string) Logger {
		return logger.New(name)
	})
	os.Exit(m.Run())
}

func TestSimpleOverseer(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	id := "echo"
	ovr.Add(id, "echo", "").Start()
	assert.True(ovr.HasProc(id))
	time.Sleep(timeUnit)

	stat := ovr.Status(id)
	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.NotEqual(stat.PID, 0, "PID shouldn't be 0")

	id = "list"
	ovr.Add(id, "ls", "/usr/").Start()
	assert.True(ovr.HasProc(id))
	time.Sleep(timeUnit)

	stat = ovr.Status(id)
	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.NotEqual(stat.PID, 0, "PID shouldn't be 0")

	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs: echo, list")
	assert.Equal(2, len(ovr.ListGroup("")), "Expected 2 procs: echo, list")

	// Should not crash
	ovr.StopAll()
}

func TestAddRemove(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	rng := make([]int, 10)
	for nr := range rng {
		assert.Equal(0, len(ovr.ListAll()))

		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)
		assert.False(ovr.Remove(id))

		assert.NotNil(ovr.Add(id, "sleep", nr))
		assert.Nil(ovr.Add(id, "sleep", "999"))

		assert.True(ovr.HasProc(id))
		assert.Equal(1, len(ovr.ListAll()))

		assert.True(ovr.Remove(id))
		assert.Equal(0, len(ovr.ListAll()))
	}
}

func TestSimpleSupervise(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	ovr.Add("echo", "echo", "")
	id := "sleep"
	assert.NotNil(ovr.Add(id, "sleep", "1"))
	assert.Nil(ovr.Add(id, "sleep", "9"))
	assert.True(ovr.HasProc(id))

	ovr.Supervise(id) // To supervise sleep. How cool is that?

	stat := ovr.Status(id)
	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.Nil(stat.Error, "Error should be nil")

	json := ovr.ToJSON(id)
	assert.Equal("finished", json.State)

	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs: echo, sleep")
}

func TestSuperviseAll(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	id := "echo"
	ovr.Add(id, "echo", "x")

	stat := ovr.Status(id)
	assert.Equal(stat.Exit, -1, "Exit code should be -1")
	assert.Equal(stat.PID, 0, "PID should be 0")

	id = "list"
	ovr.Add(id, "ls", "/usr/")

	stat = ovr.Status(id)
	assert.Equal(stat.Exit, -1, "Exit code should be 0")
	assert.Equal(stat.PID, 0, "PID should be 0")

	ovr.SuperviseAll()

	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs")

	stat = ovr.Status(id)
	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.NotEqual(stat.PID, 0, "PID should't be 0")

	json := ovr.ToJSON(id)
	assert.Equal("finished", json.State)
}

func TestSleepOverseer(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	id := "sleep"
	c := ovr.Add(id, "sleep", "10")
	c.RetryTimes = 0
	go ovr.Supervise(id)
	time.Sleep(timeUnit * 4)

	json := ovr.ToJSON(id)
	// JSON status should contain the same info
	assert.Equal("running", json.State)
	assert.Equal(-1, json.ExitCode)
	assert.True(json.PID > 0)
	assert.Nil(json.Error)

	// success kill
	assert.Nil(ovr.Signal(id, syscall.SIGKILL))
	time.Sleep(timeUnit * 4)

	// proc was killed
	json = ovr.ToJSON(id)
	assert.Equal("interrupted", json.State)
	assert.Equal(-1, json.ExitCode)
	assert.NotNil(json.Error)
}

func TestInvalidProcs(t *testing.T) {
	assert := assert.New(t)
	ovr := NewOverseer()

	ch := make(chan *ProcessJSON)
	ovr.Watch(ch)
	// ovr.UnWatch(ch)

	go func() {
		for state := range ch {
			fmt.Printf("> STATE CHANGED :: %v\n", state)
		}
	}()

	id := "err1"
	ovr.Add(id, "qwertyuiop", "zxcvbnm")
	ovr.Supervise(id)

	stat := ovr.Status(id)
	json := ovr.ToJSON(id)

	assert.Equal(stat.Exit, -1, "Exit code should be negative")
	assert.NotEqual(stat.Error, nil, "Error shouldn't be nil")
	assert.Equal("fatal", json.State)
	// JSON status should contain the same info
	assert.Equal(stat.Exit, json.ExitCode)
	assert.Equal(stat.Error, json.Error)
	assert.Equal(stat.PID, json.PID)

	// try to stop a dead process
	assert.Nil(ovr.Stop(id))

	id = "err2"
	ovr.Add(id, "ls", "/some_random_not_existent_path")
	ovr.Supervise(id)

	stat = ovr.Status(id)
	json = ovr.ToJSON(id)

	// LS returns a positive code when given a wrong path,
	// but the execution of the command overall is a success
	assert.True(stat.Exit > 0, "Exit code should be positive")
	assert.Nil(stat.Error, "Error should be nil")
	assert.Equal("finished", json.State)
}
