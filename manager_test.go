package overseer_test

// Currently using testify/assert here
// and go-test/deep for cmd_test
// Not optimal
import (
	"container/list"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	cmd "github.com/ShinyTrinkets/overseer"
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
	// The most basic Overseer test,
	// check that Add returns a Cmd
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "echo"
	ovr.Add(id, "echo").Start()
	assert.True(ovr.HasProc(id))
	time.Sleep(timeUnit)

	stat := ovr.Status(id)
	assert.Zero(stat.ExitCode)
	assert.True(stat.PID > 0)

	id = "list"
	opts := cmd.Options{Buffered: false, Streaming: false, DelayStart: 1, RetryTimes: 1}
	ovr.Add(id, "ls", []string{"/usr/"}, opts).Start()
	assert.True(ovr.HasProc(id))
	time.Sleep(timeUnit)

	stat = ovr.Status(id)
	assert.Zero(stat.ExitCode)
	assert.True(stat.PID > 0)

	assert.Equal(2, len(ovr.ListAll()), "Expected 2 procs: echo, list")
	assert.Equal(2, len(ovr.ListGroup("")), "Expected 2 procs: echo, list")

	// Should not crash
	ovr.StopAll(false)
	ovr.StopAll(true)
}

func TestOverseerNegative(t *testing.T) {
	// Negative testing. Try functions with wrong params.
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	assert.False(ovr.Remove("x"))
	assert.NotNil(ovr.Stop("x"))
	assert.NotNil(ovr.Signal("x", syscall.SIGINT))
	assert.Equal(ovr.Supervise("x"), -1)
}

func TestOverseerOptions(t *testing.T) {
	// Test all possible options when adding a proc
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
	// Test adding and removing
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	rng := make([]int, 10)
	for nr := range rng {
		assert.Zero(len(ovr.ListAll()))

		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)
		assert.False(ovr.Remove(id))

		assert.NotNil(ovr.Add(id, "sleep", []string{nr}))
		assert.Nil(ovr.Add(id, "sleep", []string{"999"}))

		assert.True(ovr.HasProc(id))
		assert.Equal(1, len(ovr.ListAll()))

		assert.True(ovr.Remove(id))
		assert.Zero(len(ovr.ListAll()))
	}
}

func TestOverseerSignalStop(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "ping"
	opts := cmd.Options{DelayStart: 0}
	ovr.Add(id, "ping", []string{"localhost"}, opts)

	stat := ovr.Status(id)
	assert.Equal("initial", stat.State)

	assert.Nil(ovr.Stop(id))
	assert.Nil(ovr.Stop(id))

	assert.Nil(ovr.Signal(id, syscall.SIGTERM))
	assert.Nil(ovr.Signal(id, syscall.SIGINT))

	stat = ovr.Status(id)
	assert.Equal("initial", stat.State)

	assert.Equal(1, len(ovr.ListAll()))

	go ovr.Supervise(id)
	time.Sleep(timeUnit)
	assert.Nil(ovr.Signal(id, syscall.SIGINT))

	go ovr.Supervise(id)
	time.Sleep(timeUnit)
	assert.Nil(ovr.Signal(id, syscall.SIGTERM))

	go ovr.Supervise(id)
	time.Sleep(timeUnit)
	assert.Nil(ovr.Stop(id))
}

func TestOverseerSupervise(t *testing.T) {
	// Test single Supervise
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
	assert.Equal(0, stat.ExitCode)
	assert.Equal("finished", stat.State)
	assert.Nil(stat.Error, "Error should be nil")

	assert.Equal([]string{"echo", "sleep"}, ovr.ListAll())
}

func TestOverseerSignalStopRestart(t *testing.T) {
	// Test stop in case of Rety > 1
	// If the retry number is a positive number,
	// the retry number is reset to 0 on cmd.Stop()
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "ping"
	opts := cmd.Options{RetryTimes: 3}
	ovr.Add(id, "ping", []string{"localhost"}, opts)

	stat := ovr.Status(id)
	assert.Equal("initial", stat.State)

	assert.Nil(ovr.Stop(id))
	assert.Nil(ovr.Signal(id, syscall.SIGTERM))

	go ovr.Supervise(id)
	time.Sleep(timeUnit)

	// Signal() doesn't reset retry times
	assert.Nil(ovr.Signal(id, syscall.SIGTERM))
	stat = ovr.Status(id)
	assert.Equal("stopping", stat.State)

	time.Sleep(timeUnit)
	stat = ovr.Status(id)
	assert.NotZero(stat.RetryTimes)
	assert.Equal("running", stat.State)

	// Stop() resets retry times
	assert.Nil(ovr.Stop(id))
	stat = ovr.Status(id)
	assert.Equal("stopping", stat.State)

	time.Sleep(timeUnit)
	stat = ovr.Status(id)
	assert.Zero(stat.RetryTimes)
	assert.Equal("interrupted", stat.State)

	ovr.StopAll(true)

	time.Sleep(timeUnit)
	stat = ovr.Status(id)
	assert.Equal("interrupted", stat.State)
}

func TestOverseerSuperviseAll(t *testing.T) {
	// Test Supervise all
	// Overseer HANGS FOREVER in this test

	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	id := "sleep"
	ovr.Add(id, "sleep", []string{"1"})

	stat := ovr.Status(id)
	assert.Equal(-1, stat.ExitCode)
	assert.Equal(0, stat.PID)

	id = "list"
	ovr.Add(id, "ls", []string{"/usr/"})

	stat = ovr.Status(id)
	assert.Equal(-1, stat.ExitCode)
	assert.Equal(0, stat.PID)

	// list before supervise
	assert.Equal([]string{"list", "sleep"}, ovr.ListAll())

	// ch1 := make(chan *cmd.ProcessJSON)
	// ovr.Watch(ch1)
	// go func() {
	// 	for state := range ch1 {
	// 		fmt.Printf("> STATE CHANGED %v\n", state)
	// 	}
	// }()

	ovr.SuperviseAll()
	// next calls shouldn't do anything
	go ovr.SuperviseAll()
	go ovr.SuperviseAll()
	go ovr.SuperviseAll()
	ovr.SuperviseAll()

	time.Sleep(time.Second + timeUnit*2)

	// list after supervise
	assert.Equal([]string{"list", "sleep"}, ovr.ListAll())

	stat = ovr.Status(id)
	assert.NotEqual(0, stat.PID)
	assert.Equal(0, stat.ExitCode)
	assert.Equal("finished", stat.State)
	pid1 := stat.PID

	// check that SuperviseAll can be run again
	ovr.SuperviseAll()

	stat = ovr.Status(id)
	pid2 := stat.PID

	assert.NotEqual(pid1, pid2)
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

	stat := ovr.Status(id)
	// JSON status should contain the same info
	assert.Equal("running", stat.State)
	assert.Equal(-1, stat.ExitCode)
	assert.NotEqual(0, stat.PID)
	assert.Nil(stat.Error)

	// success kill
	assert.Nil(ovr.Signal(id, syscall.SIGKILL))
	time.Sleep(timeUnit)

	// proc was killed
	stat = ovr.Status(id)
	assert.Equal("interrupted", stat.State)
	assert.Equal(-1, stat.ExitCode)
	assert.NotNil(stat.Error)

	// can remove now
	assert.True(ovr.Remove(id))
}

func TestOverseerWatchLogs(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	opts := cmd.Options{Buffered: false, Streaming: true}
	ovr.Add("echo", "echo", []string{"ECHO!"}, opts)
	ovr.Add("ping", "ping", []string{"127.0.0.1", "-c", "1"}, opts)

	assert.Equal(2, len(ovr.ListAll()))

	lg := make(chan *cmd.LogMsg)
	ovr.WatchLogs(lg)

	messages := ""
	lock := &sync.Mutex{}
	go func() {
		for logMsg := range lg {
			lock.Lock()
			assert.NotEqual(2, logMsg.Type) // Will this work on all platforms?
			messages += logMsg.Text
			lock.Unlock()
		}
	}()

	ovr.SuperviseAll()
	time.Sleep(timeUnit * 4)

	lock.Lock()
	assert.True(strings.ContainsAny(messages, "ECHO!"))
	assert.True(strings.ContainsAny(messages, "(127.0.0.1)"))
	assert.True(strings.ContainsAny(messages, "ping statistics"))
	lock.Unlock()
}

func TestOverseerInvalidProcs(t *testing.T) {
	assert := assert.New(t)
	ovr := cmd.NewOverseer()

	ch := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch)

	go func() {
		for state := range ch {
			fmt.Printf("> STATE CHANGED :: %v\n", state)
		}
	}()

	id := "err1"
	opts1 := cmd.Options{
		Buffered: false, Streaming: false,
		DelayStart: 2, RetryTimes: 2,
	}
	ovr.Add(id, "qwertyuiop", []string{"zxcvbnm"}, opts1)
	ovr.Supervise(id)

	stat := ovr.Status(id)

	assert.Equal(-1, stat.ExitCode, "Exit code should be negative")
	assert.NotEqual(stat.Error, nil, "Error shouldn't be nil")
	assert.Equal("fatal", stat.State)

	// try to stop a dead process
	assert.Nil(ovr.Stop(id))

	id = "err2"
	opts2 := cmd.Options{
		Buffered: false, Streaming: false,
		DelayStart: 2, RetryTimes: 2,
	}
	ovr.Add(id, "ls", []string{"/some_random_not_existent_path"}, opts2)
	ovr.Supervise(id)

	stat = ovr.Status(id)

	// LS returns a positive code when given a wrong path,
	// but the execution of the command overall is a success
	assert.True(stat.ExitCode > 0, "Exit code should be positive")
	assert.Nil(stat.Error, "Error should be nil")
	assert.Equal("finished", stat.State)
}

func TestOverseerInvalidParams(t *testing.T) {
	// The purpose of this test is to check that
	// Overseer.Add rejects invalid params
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
	// The purpose of this test is to check that
	// watch/ un-watch works as expected
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
	// un-subscribe from events
	ovr.UnWatch(ch1)

	// CH2 will receive events
	ch2 := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch2)

	ch3 := make(chan *cmd.ProcessJSON)
	ovr.Watch(ch3)
	// un-subscribe from events
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

	stat := ovr.Status(id)
	assert.Equal("finished", stat.State)

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

func TestOverseersManyInstances(t *testing.T) {
	// The purpose of this test is to check that
	// multiple Overseer instances can be run concurently
	// Also check that StopAll kills the processes
	assert := assert.New(t)
	opts := cmd.Options{
		Group: "A", Dir: "/",
		Buffered: false, Streaming: false,
		DelayStart: 1, RetryTimes: 1,
	}

	rng := []int{10, 11, 12, 13, 14}
	for _, nr := range rng {
		ovr := cmd.NewOverseer()
		assert.Equal(0, len(ovr.ListAll()))

		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)

		ovr.Add(id, "sleep", []string{nr}, opts)
		assert.Equal(1, len(ovr.ListAll()))

		go ovr.SuperviseAll()
		time.Sleep(timeUnit * 2)
		ovr.StopAll(false)
		time.Sleep(timeUnit * 2)
		ovr.StopAll(true)

		stat := ovr.Status(id)
		assert.Equal("interrupted", stat.State)
		assert.NotEqual(0, stat.PID)
	}
}

func TestOverseersExit1(t *testing.T) {
	// The purpose of this test is to check the Overseer
	// handles Exit Codes != 0
	assert := assert.New(t)
	options := []cmd.Options{
		{DelayStart: 1, RetryTimes: 1, Buffered: true},
		{DelayStart: 1, RetryTimes: 1, Streaming: true},
	}

	for _, opt := range options {
		ovr := cmd.NewOverseer()

		id := "bash1"
		ovr.Add("bash1", "bash", opt, []string{"-c", "echo 'First'; sleep 1; exit 1"})

		ovr.Supervise(id)

		stat := ovr.Status(id)
		assert.Equal(1, stat.ExitCode)
		assert.Equal(nil, stat.Error)
		assert.Equal("finished", stat.State)
	}
}

func TestOverseerKillRestart(t *testing.T) {
	// The purpose of this test is to check how Overseer
	// handles several Supervise/ Stop of the same proc
	assert := assert.New(t)
	options := []cmd.Options{
		{DelayStart: 100, RetryTimes: 1, Buffered: true},
		{DelayStart: 100, RetryTimes: 1, Streaming: true},
	}

	for _, opt := range options {
		ovr := cmd.NewOverseer()
		id := "sleep1"
		ovr.Add(id, "sleep", opt, []string{"10"})

		stat := ovr.Status(id)
		assert.Equal("initial", stat.State)
		assert.Equal(-1, stat.ExitCode)
		assert.Nil(stat.Error)

		rng := []int{1, 2, 3}
		for range rng {
			go ovr.Supervise(id)
			time.Sleep(timeUnit * 2)
			assert.Nil(ovr.Stop(id))
			time.Sleep(timeUnit * 2)

			stat = ovr.Status(id)
			assert.Equal("interrupted", stat.State)
			assert.Equal(-1, stat.ExitCode)
			assert.NotNil(stat.Error)
		}
	}
}

func TestOverseerFinishRestart(t *testing.T) {
	// The purpose of this test is to check how Overseer
	// handles several Supervise, finish, restart
	assert := assert.New(t)
	options := []cmd.Options{
		{DelayStart: 1, Buffered: true},
		{DelayStart: 1, Streaming: true},
	}

	for _, opt := range options {
		ovr := cmd.NewOverseer()

		id := "ls1"
		ovr.Add(id, "ls", opt, []string{"-la"})

		stat := ovr.Status(id)
		assert.Equal("initial", stat.State)
		assert.Equal(-1, stat.ExitCode)
		assert.Nil(stat.Error)

		rng := []int{1, 2, 3}
		for range rng {
			ovr.Supervise(id)
			time.Sleep(timeUnit * 2)
			assert.Nil(ovr.Stop(id))

			stat = ovr.Status(id)
			assert.Equal(0, stat.ExitCode)
			assert.Equal(nil, stat.Error)
			assert.Equal("finished", stat.State)
		}
		ovr.StopAll(false)
	}
}
