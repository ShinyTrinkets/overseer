<div align="center">
  <br/>
  <img src="https://raw.githubusercontent.com/ShinyTrinkets/overseer/master/logo.png" alt="Overseer logo">
  <br/>
</div>

# Overseer

[![Project name][project-img]][project-url]
[![Build status][build-img]][build-url]
[![Go Report Card][goreport-img]][goreport-url]

> Simple process manager library.

The useful methods are:

* `NewOverseer()` - Returns a new instance of an overseer manager. To register processes, use the `Add(id string, args ...string)` method, and to unregister use the `Remove(id string)` method.
* `Supervise(id string)` - Supervise a registered process and block until it finishes. This includes checking if the process was killed from outside, delaying the start and restarting in case of failure.
* `Stop(id string)` - Stop the process by sending its process group a SIGTERM signal.
* `Signal(id string)` - Signal sends an OS signal to the process group.
* `SuperviseAll()` - This is the main function. Supervise all processes and block until they finish. This includes killing all the processes when the main program exits.
* `StopAll()` - Cycle and stop all processes by sending SIGTERM.

For examples of usage, please check the tests, for now.


## Similar libraries

* https://github.com/go-cmd/cmd - os/exec.Cmd with concurrent-safe access, real-time streaming output and complete runtime/return status. Overseer is based off this one.
* https://github.com/immortal/immortal - A *nix cross-platform (OS agnostic) supervisor. The real deal.
* https://github.com/ochinchina/supervisord - A Golang supervisor implementation, inspired by Python supervisord.


Icon is made by <a href="http://www.freepik.com" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a> and licensed by <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons BY 3.0" target="_blank">CC 3.0 BY</a>.

-----

## License

[MIT](LICENSE) © Cristi Constantin.

[project-img]: https://badgen.net/badge/%E2%AD%90/Trinkets/4B0082
[project-url]: https://github.com/ShinyTrinkets
[build-img]: https://badgen.net/travis/ShinyTrinkets/overseer.go
[build-url]: https://travis-ci.org/ShinyTrinkets/overseer.go
[goreport-img]: https://goreportcard.com/badge/github.com/ShinyTrinkets/overseer.go
[goreport-url]: https://goreportcard.com/report/github.com/ShinyTrinkets/overseer.go
