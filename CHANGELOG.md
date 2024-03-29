# Overseer changelog

## v0.6
* Close STDOUT and STDERR channels
* SuperviseAll() fixes, SIGINT and SIGTERM fixes
* Fixed a CPU usage bug while supervising
* Thank you @kentlallen for all the fixes!

## v0.5
* Breaking: renamed ovr.Watch -> WatchStatus, ovr.UnWatch -> UnWatchStatus
* Using Go 1.17
* Sync from latest github.com/go-cmd/cmd
* Sync from latest github.com/jpillora/backoff
* Support for Darwin/macOS and basic support for Windows
    (all cmd tests are passing, but some manager tests are failing)

## v0.4
* Breaking: cmd.Stop() now resets RetryTimes to 0

## v0.3.4-pre
* Breaking: ovr.StopAll() function requires one bool param
* re-written Overseer procs list to use sync.Map instead of Map

## v0.3.3-pre
* Basic Windows support

## v0.3.2
* Breaking: ovr.ToJSON() is replaced with Status()
* Fixed the Buffered:true option

## v0.3.1
* Fixed Supervise bug on retry on exit code 1
* Fixed Supervise bug in case of restarting a process

## v0.3
* Breaking: renamed the repo from overseer.go to overseer
* Breaking: renamed CloneCmd function to Clone
* Breaking: removed SetEnv, SetDir, SetDelayStart, SetRetryTimes functions
* Breaking: default retries is now 0
* Moved all optional params to Options
* Added Watch and UnWatch functions for Overseer
* Added Makefile
* Added a ton of tests
* Test coverage > 90%

## v0.2
* Using Go 1.12 and modules
* Added the Overseer command line app
* Added the SetStateListener for Cmd
* Added a lot of tests

## v0.1.2
* Fixed restart delay backoff

## v0.1.1
* Cmd state fixes
* Removed ENV from Cmd JSON

## v0.1
* Initial release
