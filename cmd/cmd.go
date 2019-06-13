package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ShinyTrinkets/overseer.go"
	log "github.com/azer/logger"
	cli "github.com/jawher/mow.cli"
	quote "github.com/kballard/go-shellquote"
	yml "gopkg.in/yaml.v2"
)

// Process represents an OS process
type Process struct {
	Cmd   string   `yaml:"cmd"`
	Cwd   string   `yaml:"cwd"`
	Env   []string `yaml:"env"`
	Delay uint     `yaml:"delay"`
	Retry uint     `yaml:"retry"`
}

const (
	name    = "Overseer"
	descrip = "(<>..<>)"
)

func main() {
	overseer.SetupLogBuilder(func(name string) overseer.Logger {
		return log.New(name)
	})

	app := cli.App(name, descrip)
	app.Command("start", "Run Overseer", cmdRunAll)
	app.Run(os.Args)
}

func cmdRunAll(cmd *cli.Cmd) {
	cmd.Spec = "[-c]"
	cfgFile := cmd.StringOpt("c config", "config.yml", "the config used to define procs")

	cmd.Action = func() {
		text, err := ioutil.ReadFile(*cfgFile)
		if err != nil {
			fmt.Printf("Cannot read config. Error: %v\n", err)
			return
		}

		cfg := make(map[string]Process)
		if err := yml.Unmarshal(text, &cfg); err != nil {
			fmt.Printf("Cannot parse config. Error: %v\n", err)
			return
		}

		fmt.Println("Starting procs. Press Ctrl+C to stop...")
		ovr := overseer.NewOverseer()

		for id, proc := range cfg {
			fmt.Printf("PROC: %#v\n", proc)
			if proc.Cmd == "" {
				fmt.Printf("Proc '%s': Cmd field cannot be empty!\n", id)
				continue
			}
			args, err := quote.Split(proc.Cmd)
			if err != nil {
				fmt.Printf("Proc '%s': Cannot split args. Error: %v\n", id, err)
				continue
			}

			opts := overseer.Options{
				Buffered: false, Streaming: true,
				Env: os.Environ(),
			}
			if proc.Cwd != "" {
				opts.Dir = proc.Cwd
			}
			if proc.Delay > 0 {
				opts.DelayStart = proc.Delay
			}
			if proc.Retry > 0 {
				opts.RetryTimes = proc.Retry
			}
			p := ovr.Add(id, args[0], args[1:], opts)
			if p == nil {
				continue
			}
		}

		ovr.SuperviseAll()
		fmt.Println("\nShutdown.")
	}
}
