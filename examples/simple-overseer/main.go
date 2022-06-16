package main

import (
	"fmt"
	"time"

	cmd "github.com/ShinyTrinkets/overseer"
)

const timeUnit = 100 * time.Millisecond

// DummyLogger doesn't do anything
// The default Overseer logger is: https://github.com/ShinyTrinkets/meta-logger/blob/master/default.go
// A good production logger is: https://github.com/azer/logger/blob/master/logger.go
type DummyLogger struct {
	Name string
}

func (l *DummyLogger) Info(msg string, v ...interface{})  {}
func (l *DummyLogger) Error(msg string, v ...interface{}) {}

func main() {
	// Setup the DummyLogger here, because we capture
	// the logs above with WatchLogs()
	cmd.SetupLogBuilder(func(name string) cmd.Logger {
		return &DummyLogger{
			Name: name,
		}
	})

	ovr := cmd.NewOverseer()

	// Disable output buffering, enable streaming
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}

	// Add Cmd with options
	id1 := "ping1"
	ovr.Add(id1, "ping", []string{"localhost", "-c", "5"}, cmdOptions)

	statusFeed := make(chan *cmd.ProcessJSON)
	ovr.WatchState(statusFeed)

	// Capture status updates from the command
	go func() {
		for state := range statusFeed {
			fmt.Printf("STATE: %v\n", state)
		}
	}()

	logFeed := make(chan *cmd.LogMsg)
	ovr.WatchLogs(logFeed)

	// Capture log messages streaming from Cmd
	// The messages are already sent to log.Info or log.Error,
	// this is just a place where you can process them
	go func() {
		for log := range logFeed {
			fmt.Printf("LOG: %v\n", log)
		}
	}()

	// Run and wait for all commands to finish
	ovr.SuperviseAll()

	// Even after the command is finished, you can still access detailed info
	time.Sleep(timeUnit)
	fmt.Println(ovr.Status(id1))
}
