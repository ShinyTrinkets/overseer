# Overseer

> Simple process manager library.

The useful methods are:

* `Start(id string)` - Simply start the process and block until it finishes. It doesn't check if killed from outside.
* `Stop(id string)` - Stop the process by sending its process group a SIGTERM signal.
* `Supervise(id string)` - Supervise the process and block until it finishes. This includes checking if the process was killed from outside, delaying the start and restarting in case of failure.

-----

## License

[MIT](LICENSE) © Cristi Constantin.
