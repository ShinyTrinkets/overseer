package main

import (
	"fmt"
	"os"

	cmd "github.com/ShinyTrinkets/overseer"
)

func main() {
	ovr := cmd.NewOverseer()

	// Disable output buffering, enable streaming
	cmdOptions := cmd.Options{
		Buffered:  false,
		Streaming: true,
	}

	// Add Cmd with options
	id1 := "ping1"
	pingCmd := ovr.Add(id1, "ping", []string{"localhost", "-c", "5"}, cmdOptions)

	statusFeed := make(chan *cmd.ProcessJSON)
	ovr.Watch(statusFeed)

	// Capture status updates from the command
	go func() {
		for {
			state := <-statusFeed
			fmt.Printf("STATE: %v\n", state)
		}
	}()

	// Capture STDOUT and STDERR lines streaming from Cmd
	// If you don't capture them, they will be written into
	// the overseer log to Info or Error.
	go func() {
		for {
			select {
			case line := <-pingCmd.Stdout:
				fmt.Println(line)
			case line := <-pingCmd.Stderr:
				fmt.Fprintln(os.Stderr, line)
			}
		}
	}()

	// Run and wait for all commands to finish
	ovr.SuperviseAll()

	// Even after the command is finished, you can still access detailed info
	fmt.Println(ovr.ToJSON(id1))
}
