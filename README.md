<div align="center">
  <br/>
  <img src="https://raw.githubusercontent.com/ShinyTrinkets/overseer/master/logo.png" alt="Overseer logo">
  <br/>
</div>

# Overseer

> Simple process manager library.
> [![Go Report Card](https://goreportcard.com/badge/github.com/ShinyTrinkets/overseer)](https://goreportcard.com/report/github.com/ShinyTrinkets/overseer)

The useful methods are:

* `Start(id string)` - Simply start the process and block until it finishes. It doesn't check if killed from outside.
* `Stop(id string)` - Stop the process by sending its process group a SIGTERM signal.
* `Supervise(id string)` - Supervise the process and block until it finishes. This includes checking if the process was killed from outside, delaying the start and restarting in case of failure.

For examples of usage, please check the tests.


Icon is made by <a href="http://www.freepik.com" title="Freepik">Freepik</a> from <a href="https://www.flaticon.com/" title="Flaticon">www.flaticon.com</a> and licensed by <a href="http://creativecommons.org/licenses/by/3.0/" title="Creative Commons BY 3.0" target="_blank">CC 3.0 BY</a>.

-----

## License

[MIT](LICENSE) © Cristi Constantin.
