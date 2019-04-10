package main

import (
	"fmt"
	"io/ioutil"
	"os"

	ml "github.com/ShinyTrinkets/meta-logger"
	"github.com/ShinyTrinkets/overseer.go"
	log "github.com/azer/logger"
	cli "github.com/jawher/mow.cli"
	quote "github.com/kballard/go-shellquote"
	"gopkg.in/yaml.v2"
)

// Process represents an OS process
type Process struct {
	Cmd   string `yaml:"cmd"`
	Cwd   string `yaml:"cwd"`
	Delay uint   `yaml:"delay"`
	Retry uint   `yaml:"retry"`
}

const (
	name    = "Overseer"
	descrip = "(<>..<>)"
)

func main() {
	ml.SetupLogBuilder(func(name string) ml.Logger {
		return log.New(name)
	})

	app := cli.App(name, descrip)
	app.Command("start", "Run Overseer", cmdRunAll)
	app.Run(os.Args)
}

func cmdRunAll(cmd *cli.Cmd) {
	cmd.Spec = "-c"
	cfgFile := cmd.StringOpt("c config", "", "the config used to define procs")

	cmd.Action = func() {
		text, err := ioutil.ReadFile(*cfgFile)
		if err != nil {
			panic(err)
		}

		cfg := make(map[string]Process)
		if err := yaml.Unmarshal(text, &cfg); err != nil {
			panic(err)
		}
		fmt.Printf("%#v\n\n", cfg)

		ovr := overseer.NewOverseer()

		for id, proc := range cfg {
			if proc.Cmd == "" {
				panic("Cmd cannot be empty!")
			}
			args, err := quote.Split(proc.Cmd)
			if err != nil {
				panic(err)
			}
			p := ovr.Add(id, args...)
			if proc.Cwd != "" {
				p.SetDir(proc.Cwd)
			}
			if proc.Delay > 0 {
				p.SetDelayStart(proc.Delay)
			}
			if proc.Retry > 0 {
				p.SetRetryTimes(proc.Retry)
			}
		}

		fmt.Println("Starting procs. Press Ctrl+C to stop...")
		ovr.SuperviseAll()
		fmt.Println("\nShutdown.")
	}
}
