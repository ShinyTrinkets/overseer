# Overseer changelog

## v0.3.2
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
