<div align="center">
  <br/>
  <img src="https://raw.githubusercontent.com/ShinyTrinkets/overseer/master/logo.png" alt="Overseer logo">
  <br/>
</div>

# Overseer

[![Project name][project-img]][project-url]
[![Go Report Card][goreport-img]][goreport-url]

> Simple process manager library.

The useful methods are:

* `Stop(id string)` - Stop the process by sending its process group a SIGTERM signal.
* `Supervise(id string)` - Supervise the process and block until it finishes. This includes checking if the process was killed from outside, delaying the start and restarting in case of failure.
* `SuperviseAll()` - Supervise all processes and block until they finish. This includes killing all the processes when the main program exits.

For examples of usage, please check the tests, for now.


Icon is made by <a href="http://www.freepik.com" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a> and licensed by <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons BY 3.0" target="_blank">CC 3.0 BY</a>.

-----

## License

[MIT](LICENSE) © Cristi Constantin.

[project-img]: https://badgen.net/badge/⚙️/Trinkets/4B0082
[project-url]: https://github.com/ShinyTrinkets
[goreport-img]: https://goreportcard.com/badge/github.com/ShinyTrinkets/overseer
[goreport-url]: https://goreportcard.com/report/github.com/ShinyTrinkets/overseer
