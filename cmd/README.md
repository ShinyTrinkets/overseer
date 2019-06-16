# Overseer cli app

Small command line application that receives a YAML config file with processes and manages them.

### Usage

> overseer start -c config.yml

### Options

Each process can have the following options:

* cmd - the command to execute. The first word from the command is the executable, the rest are args.
* cwd - the working directory of the command
* env - the runtime environment for the command
* delay - delay in Millisecond applied before each run or restart (in case of failure). The overseer uses an exponential backoff, so this will increase with each restart.
* retry - the number of retries in case of failure. The default is 0 - no retries.

Config example:

```yaml
app1:
  cmd: ping localhost -c 5
  delay: 10
  retry: 5

```
