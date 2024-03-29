package overseer_test

// Currently using testify/assert here
// and go-test/deep for cmd_test
// Not optimal
import (
	"bytes"
	"container/list"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	// Test SuperviseAll many times

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

	ch1 := make(chan *cmd.ProcessJSON)
	ovr.WatchState(ch1)
	go func() {
		for state := range ch1 {
			fmt.Printf("> STATE CHANGED %v\n", state)
		}
	}()

	ovr.SuperviseAll()
	// next calls shouldn't do anything
	go ovr.SuperviseAll()
	go ovr.SuperviseAll()
	go ovr.SuperviseAll()
	ovr.SuperviseAll()

	// list after supervise
	assert.Equal([]string{"list", "sleep"}, ovr.ListAll())

	stat = ovr.Status(id)
	assert.NotEqual(0, stat.PID)
	assert.Equal(0, stat.ExitCode)
	assert.Equal("finished", stat.State)
	pid1 := stat.PID

	// check that SuperviseAll can be run again
	ovr.StopAll(false)
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

func TestOverseerWatchLogsSupervise(t *testing.T) {
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

	ovr.Supervise("echo")
	ovr.Supervise("ping")
	// for some stupid reason, this wait is needed for CI
	time.Sleep(timeUnit)

	lock.Lock()
	assert.True(strings.ContainsAny(messages, "ECHO!"))
	assert.True(strings.ContainsAny(messages, "(127.0.0.1)"))
	assert.True(strings.ContainsAny(messages, "ping statistics"))
	lock.Unlock()
}

func TestOverseerWatchLogsSuperviseAll(t *testing.T) {
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
	// for some stupid reason, this wait is needed for CI
	time.Sleep(timeUnit)

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
	ovr.WatchState(ch)

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
	// watch/ un-watch state works as expected
	// For some unknown and idiotic reason, this test fails on CI
	// I tried to reproduce it locally on Linux and macOS, but I can't
	// Even in a local Docker container with Ubuntu, the test passes

	// skipping for now ...
	t.Skip("FIXME")

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
	ovr.WatchState(ch1)
	// un-subscribe from events
	ovr.UnWatchState(ch1)

	// CH2 will receive events
	ch2 := make(chan *cmd.ProcessJSON)
	ovr.WatchState(ch2)

	ch3 := make(chan *cmd.ProcessJSON)
	ovr.WatchState(ch3)
	// un-subscribe from events
	ovr.UnWatchState(ch3)

	// CH4 will also receive events
	ch4 := make(chan *cmd.ProcessJSON)
	ovr.WatchState(ch4)

	go func() {
		for {
			select {
			case state := <-ch1:
				fmt.Printf("> STATE WRONG 1 %v\n", state)
				pushResult(state.State)
			case state := <-ch2:
				fmt.Printf("> STATE CHANGED 2 %v\n", state)
				pushResult(state.State)
			case state := <-ch3:
				fmt.Printf("> STATE WRONG 3 %v\n", state)
				pushResult(state.State)
			case state := <-ch4:
				fmt.Printf("> STATE CHANGED 4 %v\n", state)
				pushResult(state.State)
			}
		}
	}()

	id := "date"
	ovr.Add(id, "date", []string{"+%Y-%m-%d %H:%M:%S"})
	ovr.SuperviseAll()

	stat := ovr.Status(id)
	assert.Equal("finished", stat.State)

	lock.Lock()
	assert.Equal(6, results.Len())
	// starting, running, finished x 2
	assert.Equal("starting", results.Front().Value)
	assert.Equal("finished", results.Back().Value)
	lock.Unlock()
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
		time.Sleep(timeUnit)
		ovr.StopAll(false)
		time.Sleep(timeUnit)
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
			time.Sleep(timeUnit)
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
			time.Sleep(timeUnit)
			assert.Nil(ovr.Stop(id))

			stat = ovr.Status(id)
			assert.Equal(0, stat.ExitCode)
			assert.Equal(nil, stat.Error)
			assert.Equal("finished", stat.State)
		}
		ovr.StopAll(false)
	}
}

func TestOverseerStreamingCpuUsageManual(t *testing.T) {
	// Use 'htop', 'System Monitor' or similar application to monitor cpu usage.
	t.Skip("Use this to manually monitor CPU usage. Skip by default")
	opts := cmd.Options{
		Dir:       "/tmp",
		Streaming: true,
	}

	var ovr *cmd.Overseer
	var wg sync.WaitGroup

	ovr = cmd.NewOverseer()

	allCmds := map[string]*cmd.Cmd{}
	for nr := range [10]int{} {
		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)
		c := ovr.Add(id, "tail", []string{"-F", "/dev/null"}, opts)
		allCmds[string(nr)] = c
	}

	for _, c := range allCmds {
		wg.Add(2)
		go func(c *cmd.Cmd) {
			defer wg.Done()

			for stdOut := range c.Stdout {
				fmt.Printf("%s\n", stdOut)
			}
			fmt.Printf("c.Stdout finished \n")
		}(c)
		go func(c *cmd.Cmd) {
			defer wg.Done()

			for stdErr := range c.Stderr {
				fmt.Printf("%s\n", stdErr)
			}
			fmt.Printf("c.Stderr finished\n")
		}(c)
	}

	watchOut := make(chan *cmd.ProcessJSON, 50)
	ovr.WatchState(watchOut)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for status := range watchOut {
			fmt.Printf("%s\n", status.State)
		}
		fmt.Printf("WatchState finished\n")
	}()

	ovrLogOut := make(chan *cmd.LogMsg, 50)
	ovr.WatchLogs(ovrLogOut)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for msg := range ovrLogOut {
			fmt.Printf("%s\n", msg.Text)
		}
		fmt.Printf("WatchLogs finished\n")
	}()

	go func() {
		fmt.Println("Starting Supervise")
		ovr.SuperviseAll()
		fmt.Println("Done Supervise")
	}()

	time.Sleep(6 * time.Second)

	fmt.Println("Sending Soft Stop Command")
	ovr.StopAll(false)

	for name, c := range allCmds {
		if c.IsFinalState() {
			ovr.Remove(name)
			delete(allCmds, name)
		}
	}

	time.Sleep(6 * time.Second)

	fmt.Println("Sending Hard Stop Command")
	ovr.StopAll(true)

	for name, c := range allCmds {
		if c.IsFinalState() {
			ovr.Remove(name)
			delete(allCmds, name)
		}
	}

	fmt.Println("Unwatching Logs")
	ovr.UnWatchState(watchOut)
	ovr.UnWatchLogs(ovrLogOut)
	close(watchOut)
	close(ovrLogOut)

	fmt.Println("Sleeping for 60 seconds to give extra time for CPU monitoring")
	time.Sleep(60 * time.Second)

	fmt.Println("Waiting For log goroutines to finish.")
	wg.Wait()
	fmt.Println("Done. Commands should be finished.")
}

// This test supports an optional environment variable to define the
// upper limit of CPU usage for this test process.
// Use:
// CPU_LIMIT=300 go test -v -race -run TestOverseerStreamingCpuUsageAuto
func TestOverseerStreamingCpuUsageAuto(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("CPU usage test only supported on Linux. Use TestOverseerStreamingCpuUsageManual")
	}
	assert := assert.New(t)

	var testDone atomic.Bool
	testDone.Store(false)

	// Get user defined CPU_LIMIT if provided
	cpuLimitStr, ok := os.LookupEnv("CPU_LIMIT")
	if !ok {
		// Before CPU fix, usage would surpass 1000% on some systems
		// From all 10 procs started in this test hitting 100%
		cpuLimitStr = "100"
	}
	cpuLimit := parseFloat(cpuLimitStr)

	// Goroutine to monitor CPU usage. Fail test if CPU usage > cpuLimit or 100% if not defined
	pid := os.Getpid()
	go func() {
		for {
			history = make(map[int]Stat)
			cpuUsagePct := statFromProc(pid)
			// If failing here you might need to set CPU_LIMIT env var
			assert.LessOrEqual(cpuUsagePct, cpuLimit)
			fmt.Printf("PID %d CPU Usage: %.2f%%\n", pid, cpuUsagePct)
			time.Sleep(1 * time.Second)
			if testDone.Load() {
				return
			}
		}
	}()

	opts := cmd.Options{
		Dir:       "/tmp",
		Streaming: true,
	}

	var ovr *cmd.Overseer
	var wg sync.WaitGroup

	ovr = cmd.NewOverseer()

	// Add 10 commands to overseer
	allCmds := map[string]*cmd.Cmd{}
	for nr := range [10]int{} {
		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)
		c := ovr.Add(id, "tail", []string{"-F", "/dev/null"}, opts)
		allCmds[string(nr)] = c
	}

	for _, c := range allCmds {
		wg.Add(2)
		// Iterate over cmd's Stdout
		go func(c *cmd.Cmd) {
			defer wg.Done()
			for range c.Stdout {
			}
		}(c)
		// Iterate over cmd's Stderr
		go func(c *cmd.Cmd) {
			defer wg.Done()
			for range c.Stderr {
			}
		}(c)
	}

	// Watch state
	watchOut := make(chan *cmd.ProcessJSON, 50)
	ovr.WatchState(watchOut)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range watchOut {
		}
	}()

	// Watch logs
	ovrLogOut := make(chan *cmd.LogMsg, 50)
	ovr.WatchLogs(ovrLogOut)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range ovrLogOut {
		}
	}()

	go func() {
		ovr.SuperviseAll()
	}()

	time.Sleep(1 * time.Second)

	ovr.StopAll(false)

	for name, c := range allCmds {
		if c.IsFinalState() {
			ovr.Remove(name)
			delete(allCmds, name)
		}
	}

	time.Sleep(1 * time.Second)

	ovr.StopAll(true)

	for name, c := range allCmds {
		if c.IsFinalState() {
			ovr.Remove(name)
			delete(allCmds, name)
		}
	}

	ovr.UnWatchState(watchOut)
	ovr.UnWatchLogs(ovrLogOut)
	close(watchOut)
	close(ovrLogOut)

	time.Sleep(3 * time.Second)

	wg.Wait()
	testDone.Store(true)
}

func TestOverseerProcsStreaming(t *testing.T) {
	// The purpose of this test is to check if all
	// overseer processes stop after all
	// managed procs have stopped
	if runtime.GOOS != "linux" {
		t.Skip("This test is only supported on Linux.")
	}

	assert := assert.New(t)
	assert.Equal(0, len(getOverseerProcs()), "Overseer procs exist from other tests. There may be a bug.")

	opts := cmd.Options{
		Group: "A", Dir: "/",
		Buffered: false, Streaming: true,
		DelayStart: 1, RetryTimes: 1,
	}

	ovr := cmd.NewOverseer()
	assert.Equal(0, len(ovr.ListAll()))

	ovr.Add("id1", "sleep", []string{"10"}, opts)
	assert.Equal(1, len(ovr.ListAll()))

	go ovr.SuperviseAll()

	// Give some time for cmds to start
	time.Sleep(timeUnit)

	// Should be at least 1 proc/overseer and 1 proc/cmd
	assert.GreaterOrEqual(len(getOverseerProcs()), 2)

	ovr.StopAll(true)

	// Give some time for everything to close
	time.Sleep(timeUnit)

	stat := ovr.Status("id1")
	assert.Equal("interrupted", stat.State)
	assert.NotEqual(0, stat.PID)
	assert.Equal(0, len(getOverseerProcs()))
}

func TestOverseerProcsBuffered(t *testing.T) {
	// The purpose of this test is to check if all
	// overseer processes stop after all
	// managed procs have stopped
	if runtime.GOOS != "linux" {
		t.Skip("This test is only supported on Linux.")
	}

	assert := assert.New(t)
	assert.Equal(0, len(getOverseerProcs()), "Overseer procs exist from other tests. There may be a bug.")

	opts := cmd.Options{
		Group: "A", Dir: "/",
		Buffered: true, Streaming: false,
		DelayStart: 1, RetryTimes: 1,
	}

	ovr := cmd.NewOverseer()
	assert.Equal(0, len(ovr.ListAll()))

	ovr.Add("id1", "sleep", []string{"10"}, opts)
	assert.Equal(1, len(ovr.ListAll()))

	go ovr.SuperviseAll()

	// Give some time for cmds to start
	time.Sleep(timeUnit)

	// Should be at least 1 proc/overseer and 1 proc/cmd
	assert.GreaterOrEqual(len(getOverseerProcs()), 2)

	ovr.StopAll(true)

	// Give some time for everything to close
	time.Sleep(timeUnit)

	stat := ovr.Status("id1")
	assert.Equal("interrupted", stat.State)
	assert.NotEqual(0, stat.PID)
	assert.Equal(0, len(getOverseerProcs()))
}

func TestOverseerProcsManyInstancesNoOutput(t *testing.T) {
	// The purpose of this test is to check if all
	// overseer processes stop after all managed procs
	// have stopped when no output is selected
	// and when multiple instances of overseer exist
	if runtime.GOOS != "linux" {
		t.Skip("This test is only supported on Linux.")
	}

	assert := assert.New(t)
	assert.Equal(0, len(getOverseerProcs()), "Overseer procs exist from other tests. There may be a bug.")

	opts := cmd.Options{
		Group: "A", Dir: "/",
		Buffered: false, Streaming: false,
		DelayStart: 1, RetryTimes: 1,
	}

	ovrs := map[string]*cmd.Overseer{}

	rng := []int{10, 11, 12, 13, 14}
	for _, nr := range rng {
		ovr := cmd.NewOverseer()
		assert.Equal(0, len(ovr.ListAll()))

		nr := strconv.Itoa(nr)
		id := fmt.Sprintf("id%s", nr)

		ovr.Add(id, "sleep", []string{nr}, opts)
		assert.Equal(1, len(ovr.ListAll()))

		ovrs[id] = ovr
	}

	// Should be no overseer procs as SuperviseAll hasn't been called
	assert.Equal(len(getOverseerProcs()), 0)

	for _, ovr := range ovrs {
		go ovr.SuperviseAll()
	}

	// Give some time for cmds to start
	time.Sleep(timeUnit)

	// Should be at least 1 proc/overseer and 1 proc/cmd
	assert.GreaterOrEqual(len(getOverseerProcs()), len(ovrs)*2)

	for _, ovr := range ovrs {
		ovr.StopAll(true)
	}

	// Give some time for everything to close
	time.Sleep(timeUnit)

	for id, ovr := range ovrs {
		stat := ovr.Status(id)
		assert.Equal("interrupted", stat.State)
		assert.NotEqual(0, stat.PID)
	}
	assert.Equal(0, len(getOverseerProcs()))
}

func TestOverseerSystemSIGINT(t *testing.T) {
	// The purpose of this test is to ensure
	// SIGINT is handled properly and all
	// overseer procs are stopped gracefully
	if runtime.GOOS != "linux" {
		t.Skip("This test is only supported on Linux.")
	}
	assert := assert.New(t)

	opts := cmd.Options{
		Group: "A", Dir: "/",
		Buffered: false, Streaming: true,
		DelayStart: 1, RetryTimes: 1,
	}

	ovr := cmd.NewOverseer()
	assert.Equal(0, len(ovr.ListAll()))

	ovr.Add("id1", "sleep", []string{"10"}, opts)
	assert.Equal(1, len(ovr.ListAll()))

	lg := make(chan *cmd.LogMsg)
	ovr.WatchLogs(lg)

	messages := ""
	lock := &sync.Mutex{}
	go func() {
		for logMsg := range lg {
			lock.Lock()
			messages += logMsg.Text
			lock.Unlock()
		}
	}()

	go ovr.SuperviseAll()

	// Give some time for cmds to start
	time.Sleep(timeUnit)

	// Should be at least 1 proc/overseer and 1 proc/cmd
	assert.GreaterOrEqual(len(getOverseerProcs()), 2)

	// Send SIGINT to PID
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)

	// Give some time for everything to close
	time.Sleep(timeUnit)

	stat := ovr.Status("id1")
	assert.Equal("interrupted", stat.State)
	assert.NotEqual(0, stat.PID)
	assert.Equal(0, len(getOverseerProcs()))
	lock.Lock()
	assert.True(strings.ContainsAny(messages, "Received signal: map[sig:interrupt]!"))
	lock.Unlock()
}

func TestOverseerSystemSIGTERM(t *testing.T) {
	// The purpose of this test is to ensure
	// SIGTERM is handled properly and all
	// overseer procs are stopped gracefully
	assert := assert.New(t)

	// assert.Equal(0, len(getOverseerProcs()), "Overseer procs exist from other tests. There may be a bug.")

	opts := cmd.Options{
		Group: "A", Dir: "/",
		Buffered: false, Streaming: true,
		DelayStart: 1, RetryTimes: 1,
	}

	ovr := cmd.NewOverseer()
	assert.Equal(0, len(ovr.ListAll()))

	ovr.Add("id1", "sleep", []string{"10"}, opts)
	assert.Equal(1, len(ovr.ListAll()))

	lg := make(chan *cmd.LogMsg)
	ovr.WatchLogs(lg)

	messages := ""
	lock := &sync.Mutex{}
	go func() {
		for logMsg := range lg {
			lock.Lock()
			messages += logMsg.Text
			lock.Unlock()
		}
	}()

	go ovr.SuperviseAll()

	// Give some time for cmds to start
	time.Sleep(timeUnit)

	// Should be at least 1 proc/overseer and 1 proc/cmd
	assert.GreaterOrEqual(len(getOverseerProcs()), 2)

	// Send SIGTERM to PID
	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	// Give some time for everything to close
	time.Sleep(timeUnit)

	stat := ovr.Status("id1")
	assert.Equal("interrupted", stat.State)
	assert.NotEqual(0, stat.PID)
	assert.Equal(0, len(getOverseerProcs()))
	lock.Lock()
	assert.True(strings.ContainsAny(messages, "Received signal: map[sig:terminated]!"))
	lock.Unlock()
}

/**********************************************************************************/
// Test helper functions

// Returns stack trace for Overseer goroutines
func getOverseerProcs() []string {
	overseerProcs := []string{}

	// Get all goroutine stack traces
	profile1 := pprof.Lookup("goroutine")
	var buff bytes.Buffer
	profile1.WriteTo(&buff, 2)

	// Split by 'paragraph' which is the full stack trace for a single goroutine
	re := regexp.MustCompile(`\r?\n\n`)
	splitStr := re.Split(buff.String(), -1) // -1 to return all substrings

	// Extract only overseer goroutines that aren't test processes
	for _, line := range splitStr {
		if strings.Contains(line, "overseer") && !strings.Contains(line, "test") {
			overseerProcs = append(overseerProcs, line)
		}
	}
	return overseerProcs
}

/**********************************************************************************/
// PID CPU Monitor
// adapted from https://github.com/struCoder/pidusage
type Stat struct {
	utime  float64
	stime  float64
	cutime float64
	cstime float64
	start  float64
	uptime float64
}

func formatStdOut(stdout []byte, userfulIndex int) []string {
	infoArr := strings.Split(string(stdout), "\n")[userfulIndex]
	ret := strings.Fields(infoArr)
	return ret
}

func parseFloat(val string) float64 {
	floatVal, _ := strconv.ParseFloat(val, 64)
	return floatVal
}

var history map[int]Stat
var historyLock sync.Mutex

func statFromProc(pid int) float64 {
	clkTckStdout, err := exec.Command("getconf", "CLK_TCK").Output()
	var clkTck float64
	if err == nil {
		clkTck = parseFloat(formatStdOut(clkTckStdout, 0)[0])
	}

	uptimeFileBytes, err := ioutil.ReadFile(path.Join("/proc", "uptime"))
	if err != nil {
		return math.MaxFloat64
	}
	uptime := parseFloat(strings.Split(string(uptimeFileBytes), " ")[0])

	procStatFileBytes, err := ioutil.ReadFile(path.Join("/proc", strconv.Itoa(pid), "stat"))
	if err != nil {
		return math.MaxFloat64
	}
	splitAfter := strings.SplitAfter(string(procStatFileBytes), ")")

	if len(splitAfter) == 0 || len(splitAfter) == 1 {
		fmt.Printf("Can't find process with this PID: %d\n", pid)
		return math.MaxFloat64
	}
	infos := strings.Split(splitAfter[1], " ")
	stat := &Stat{
		utime:  parseFloat(infos[12]),
		stime:  parseFloat(infos[13]),
		cutime: parseFloat(infos[14]),
		cstime: parseFloat(infos[15]),
		start:  parseFloat(infos[20]) / clkTck,
		uptime: uptime,
	}

	_stime := 0.0
	_utime := 0.0

	historyLock.Lock()
	defer historyLock.Unlock()

	_history := history[pid]

	if _history.stime != 0 {
		_stime = _history.stime
	}

	if _history.utime != 0 {
		_utime = _history.utime
	}
	total := stat.stime - _stime + stat.utime - _utime
	total = total / clkTck

	seconds := stat.start - uptime
	if _history.uptime != 0 {
		seconds = uptime - _history.uptime
	}

	seconds = math.Abs(seconds)
	if seconds == 0 {
		seconds = 1
	}

	history[pid] = *stat
	return (total / seconds) * 100
}

/**********************************************************************************/
