package overseer_test

// Currently using testify/assert here
// and go-test/deep for cmd_test
// Not optimal
import (
	"container/list"
	"fmt"
	"os"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	cmd "github.com/ShinyTrinkets/overseer.go"
	log "github.com/azer/logger"
	"github.com/stretchr/testify/assert"
)

const timeUnit = 200 * time.Millisecond

func TestMain(m *testing.M) {
	cmd.SetupLogBuilder(func(name string) cmd.Logger {
		return log.New(name)
	})
	os.Exit(m.Run())
}

func TestOverseerBasic(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "echo"
	ovr.Add(id, "echo").Start()
	assert.True(ovr.HasProc(id))
	time.Sleep(timeUnit)

	stat := ovr.Status(id)
	assert.Equal(0, stat.Exit)
	assert.True(stat.PID > 0)

	id = "list"
	opts := cmd.Options{Buffered: false, Streaming: false, DelayStart: 1, RetryTimes: 1}
	ovr.Add(id, "ls", []string{"/usr/"}, opts).Start()
	assert.True(ovr.HasProc(id))
	time.Sleep(timeUnit)

	stat = ovr.Status(id)
	assert.Equal(0, stat.Exit)
	assert.True(stat.PID > 0)

	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs: echo, list")
	assert.Equal(2, len(ovr.ListGroup("")), "Expected 2 procs: echo, list")

	// Should not crash
	ovr.StopAll()
}

func TestOverseerOptions(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "ls"
	opts := cmd.Options{
		Group: "A", Dir: "/", Env: []string{"YES"},
		Buffered: false, Streaming: false,
		DelayStart: 1, RetryTimes: 9,
	}

	c := ovr.Add(id, "ls", opts)

	assert.Equal(c.Group, opts.Group)
	assert.Equal(c.Dir, opts.Dir)
	assert.Equal(c.Env, opts.Env)
	assert.Equal(c.DelayStart, opts.DelayStart)
	assert.Equal(c.RetryTimes, opts.RetryTimes)
}

func TestOverseerAddRemove(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	rng := make([]int, 10)
	for nr := range rng {
		assert.Equal(0, len(ovr.ListAll()))

		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)
		assert.False(ovr.Remove(id))

		assert.NotNil(ovr.Add(id, "sleep", []string{nr}))
		assert.Nil(ovr.Add(id, "sleep", []string{"999"}))

		assert.True(ovr.HasProc(id))
		assert.Equal(1, len(ovr.ListAll()))

		assert.True(ovr.Remove(id))
		assert.Equal(0, len(ovr.ListAll()))
	}
}

func TestOverseerSignalStop(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "ping"
	opts := cmd.Options{Buffered: false, Streaming: false, DelayStart: 1}
	ovr.Add(id, "ping", []string{"localhost"}, opts)

	json := ovr.ToJSON(id)
	assert.Equal("initial", json.State)

	assert.Nil(ovr.Stop(id))
	assert.Nil(ovr.Stop(id))

	assert.Nil(ovr.Signal(id, syscall.SIGTERM))
	assert.Nil(ovr.Signal(id, syscall.SIGINT))

	json = ovr.ToJSON(id)
	assert.Equal("initial", json.State)

	assert.Equal(1, len(ovr.ListAll()))
}

func TestOverseerSupervise(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	opts := cmd.Options{Buffered: false, Streaming: false}
	ovr.Add("echo", "echo", opts)
	id := "sleep"
	// double add test
	assert.NotNil(ovr.Add(id, "sleep", []string{"1"}))
	assert.Nil(ovr.Add(id, "sleep", []string{"9"}))

	ovr.Supervise(id) // To supervise sleep. How cool is that?

	stat := ovr.Status(id)
	assert.Equal(stat.Exit, 0, "Exit code should be 0")
	assert.Nil(stat.Error, "Error should be nil")

	json := ovr.ToJSON(id)
	assert.Equal("finished", json.State)

	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs: echo, sleep")
}

func TestOverseerSuperviseAll(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "echo"
	ovr.Add(id, "echo", []string{"x"})

	stat := ovr.Status(id)
	assert.Equal(stat.Exit, -1, "Exit code should be -1")
	assert.Equal(stat.PID, 0)

	id = "list"
	ovr.Add(id, "ls", []string{"/usr/"})

	// before supervise
	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs")

	stat = ovr.Status(id)
	assert.Equal(-1, stat.Exit)
	assert.Equal(0, stat.PID)

	// block and wait for procs to finish
	go ovr.SuperviseAll()
	// next calls shouldn't do anything
	ovr.SuperviseAll()
	ovr.SuperviseAll()

	time.Sleep(timeUnit * 4)

	// after supervise
	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs")

	stat = ovr.Status(id)
	assert.Equal(0, stat.Exit)
	assert.NotEqual(0, stat.PID)

	json := ovr.ToJSON(id)
	assert.Equal("finished", json.State)
}

func TestOverseerSleep(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "sleep"
	opts := cmd.Options{Buffered: false, Streaming: false, DelayStart: 1}
	ovr.Add(id, "sleep", []string{"10"}, opts)
	go ovr.Supervise(id)
	time.Sleep(timeUnit)

	// can't remove while it's running
	assert.False(ovr.Remove(id))

	json := ovr.ToJSON(id)
	// JSON status should contain the same info
	assert.Equal("running", json.State)
	assert.Equal(-1, json.ExitCode)
	assert.NotEqual(0, json.PID)
	assert.Nil(json.Error)

	// success kill
	assert.Nil(ovr.Signal(id, syscall.SIGKILL))
	time.Sleep(timeUnit)

	// proc was killed
	json = ovr.ToJSON(id)
	assert.Equal("interrupted", json.State)
	assert.Equal(-1, json.ExitCode)
	assert.NotNil(json.Error)

	// can remove now
	assert.True(ovr.Remove(id))
}

func TestOverseerInvalidProcs(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	ch := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch)
	// ovr.UnWatch(ch)

	go func() {
		for state := range ch {
			fmt.Printf("> STATE CHANGED :: %v\n", state)
		}
	}()

	id := "err1"
	ovr.Add(id, "qwertyuiop", []string{"zxcvbnm"})
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
	ovr.Add(id, "ls", []string{"/some_random_not_existent_path"})
	ovr.Supervise(id)

	stat = ovr.Status(id)
	json = ovr.ToJSON(id)

	// LS returns a positive code when given a wrong path,
	// but the execution of the command overall is a success
	assert.True(stat.Exit > 0, "Exit code should be positive")
	assert.Nil(stat.Error, "Error should be nil")
	assert.Equal("finished", json.State)
}

func TestOverseerInvalidParams(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	assert.NotNil(ovr.Add("valid1", "x", []string{"abc"}))
	assert.NotNil(ovr.Add("valid2", "y", cmd.Options{Buffered: false, Streaming: false}))
	// empty exec command
	assert.Nil(ovr.Add("err1", ""))
	// invalid optional param
	assert.Nil(ovr.Add("err2", "x", 1))
	assert.Nil(ovr.Add("err3", "x", nil))
	assert.Nil(ovr.Add("err4", "x", true))
}

func TestOverseerWatchUnwatch(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	lock := &sync.Mutex{}
	results := list.New()
	pushResult := func(result string) {
		lock.Lock()
		results.PushBack(result)
		lock.Unlock()
	}

	ch1 := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch1)
	ovr.UnWatch(ch1)

	// CH2 will receive events
	ch2 := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch2)

	ch3 := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch3)
	ovr.UnWatch(ch3)

	// CH4 will also receive events
	ch4 := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch4)

	go func() {
		for {
			select {
			case state := <-ch1:
				fmt.Printf("> STATE CHANGED 1 %v\n", state)
				pushResult(state.State)
			case state := <-ch2:
				fmt.Printf("> STATE CHANGED 2 %v\n", state)
				pushResult(state.State)
			case state := <-ch3:
				fmt.Printf("> STATE CHANGED 3 %v\n", state)
				pushResult(state.State)
			case state := <-ch4:
				fmt.Printf("> STATE CHANGED 4 %v\n", state)
				pushResult(state.State)
			}
		}
	}()

	id := "ls"
	ovr.Add(id, "ls", []string{"-la"})
	ovr.SuperviseAll()

	json := ovr.ToJSON(id)
	assert.Equal("finished", json.State)

	lock.Lock()
	assert.Equal(6, results.Len())
	// starting, running, finished x 2
	assert.Equal("starting", results.Front().Value)
	assert.Equal("finished", results.Back().Value)
	lock.Unlock()

	// // Iterate through statuses and print its contents
	// for e := results.Front(); e != nil; e = e.Next() {
	// 	fmt.Println(e.Value)
	// }
}
