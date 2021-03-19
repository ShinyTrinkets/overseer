<div align="center">
  <br/>
  <img src="https://raw.githubusercontent.com/ShinyTrinkets/overseer/master/logo.png" alt="Overseer logo">
  <br/>
</div>

# Overseer

[![Project name][project-img]][project-url]
[![Build status][build-img]][build-url]
[![Coverage report][cover-img]][cover-url]
[![Go Report Card][goreport-img]][goreport-url]
[![Go Reference][goref-img]][goref-url]

> Simple process manager library.


* **Note**: The master branch is the development branch. To make sure you use the correct version, use the repository tags.

At the heart of this library is the [os/exec.Cmd](https://golang.org/pkg/os/exec/#Cmd) from Go-lang and the first wrapper for that is the **Cmd struct**.<br/>
The **Overseer struct** can supervise one or more Cmds running at the same time.<br/>
You can safely run multiple Overseer instances at the same time.

There are 3 states in the normal lifecycle of a proc: *starting, running, finished*.<br/>
If the process is killed prematurely, the states are: *starting, running, interrupted*.<br/>
If the process cannot start, the states are: *starting, fatal*.


## Overseer API

Setting up a logger is optional, but if you want to use it, it must be called before creating a new Overseer.<br/>
By default, the logger is `DefaultLogger` from [ShinyTrinkets/meta-logger/default.go](https://github.com/ShinyTrinkets/meta-logger/blob/master/default.go).<br/>
To disable the logger completely, you need to create a Logger interface (with functions Info and Error) that don't do anything.

* `NewOverseer()` - Returns a new instance of the Overseer process manager.
* `Add(id string, exec string, args ...interface{})` - Register a proc, without starting it. The `id` must be unique. The name of the executable is `exec`. The args of the executable are `args`.
* `Remove(id string)` - Unregister a proc, only if it's not running. The `id` must be unique.
* `SuperviseAll()` - This is *the main function*. Supervise all registered processes and block until they finish. This includes killing all the processes when the main program exits. The function can be called again, after all the processes are finished. The status of the running processes can be watched live with the `Watch()` function.
* `Supervise(id string)` - Supervise one registered process and block until it finishes. This includes checking if the process was killed from the outside, delaying the start and restarting in case of failure (failure means the program has an exit code != 0 or it ran with errors). The function can be called again, after the process is finished.
* `Watch(outputChan chan *ProcessJSON)` - Subscribe to all state changes via the provided output channel. The channel will receive status changes for all the added procs, but you can easily identify the one your are interested in from the ID, Group, etc. Note that for each proc you will receive only 2 or 3 messages that represent all the possible states (eg: starting, running, finished).
* `UnWatch(outputChan chan *ProcessJSON)` - Un-subscribe from the state changes, by un-registering the channel.
* `Stop(id string)` - Stops the process by sending its process group a SIGTERM signal and resets RetryTimes to 0 so the process doesn't restart.
* `Signal(id string, sig syscall.Signal)` - Sends an OS signal to the process group.
* `StopAll(kill bool)` - Cycles and stops all processes. If "kill" is false, all procs receive SIGTERM to allow a graceful shut down. If "kill" is true, all procs receive SIGKILL and they are killed immediately.

## Cmd API

It's recommended to use the higher level Overseer, instead of Cmd directly.<br/>
If you use Cmd directly, keep in mind that it is *one use only*. After starting a instance, it cannot be started again. However, you can `Clone` your instance and start the clone. The `Supervise` method from the Overseer does all of that for you.

* `NewCmd(name string, args ...interface{})` - Returns a new instance of Cmd.
* `Clone()` - Clones a Cmd. All the options are copied, but the state of the original object is lost.
* `Start()` - Starts the command and immediately returns a channel that the caller can use to receive the final Status of the command when it ends. The function **can only be called once**.
* `Stop()` - Stops the command by sending its process group a SIGTERM signal.
* `Signal(sig syscall.Signal)` - Sends an OS signal to the process group.
* `Status()` - Returns the Status of the command at any time. The Status struct contains: PID, Exit code, Error (if it's the case) Start and Stop timestamps, Runtime in seconds.
* `IsInitialState()` - true if the Cmd is in initial state.
* `IsRunningState()` - true if the Cmd is starting, or running.
* `IsFinalState()` - true if the Cmd is in a final state.


## Project highlights

* real-time status
* real-time stdout and stderr
* complete and consolidated return
* easy to track process state
* proper process termination on program exit
* portable command line binary for managing procs
* heavily tested, very good test coverage
* no race conditions


For examples of usage, please check the [Examples](examples/) folder, the [manager tests](manager_test.go), the [Overseer command line app](cmd/cmd.go), or the [Spinal app](https://github.com/ShinyTrinkets/spinal/blob/master/main.go).


## Similar libraries

* https://github.com/go-cmd/cmd - os/exec.Cmd with concurrent-safe access, real-time streaming output and complete runtime/return status. Overseer is based off this one.
* https://github.com/immortal/immortal - A *nix cross-platform (OS agnostic) supervisor. The real deal.
* https://github.com/ochinchina/supervisord - A Golang supervisor implementation, inspired by Python supervisord.
* https://github.com/DarthSim/hivemind - Process manager for Procfile-based applications.


Icon is made by <a href="http://www.freepik.com" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a> and licensed by <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons BY 3.0" target="_blank">CC 3.0 BY</a>.

-----

## License

[MIT](LICENSE) © Cristi Constantin.

[project-img]: https://badgen.net/badge/%E2%AD%90/Trinkets/4B0082
[project-url]: https://github.com/ShinyTrinkets
[build-img]: https://badgen.net/travis/ShinyTrinkets/overseer
[build-url]: https://travis-ci.org/ShinyTrinkets/overseer
[cover-img]: https://codecov.io/gh/ShinyTrinkets/overseer/branch/master/graph/badge.svg
[cover-url]: https://codecov.io/gh/ShinyTrinkets/overseer
[goreport-img]: https://goreportcard.com/badge/github.com/ShinyTrinkets/overseer
[goreport-url]: https://goreportcard.com/report/github.com/ShinyTrinkets/overseer
[goref-img]: https://pkg.go.dev/badge/github.com/ShinyTrinkets/overseer.svg
[goref-url]: https://pkg.go.dev/github.com/ShinyTrinkets/overseer
