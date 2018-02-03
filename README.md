# GoRcon-ArmA
The Go based Rcon solution for ArmA servers with server management features

[![build status](https://git.play-net.org/playnet-public/gorcon-arma/badges/master/build.svg)](https://git.play-net.org/playnet-public/gorcon-arma/commits/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/playnet-public/gorcon-arma)](https://goreportcard.com/report/github.com/playnet-public/gorcon-arma)
[![GitHub license](https://img.shields.io/badge/license-AGPL-blue.svg)](https://raw.githubusercontent.com/playnet-public/gorcon-arma/master/LICENSE)
[![Join Discord at https://discord.gg/dWZkR6R](https://img.shields.io/badge/style-join-green.svg?style=flat&label=Discord)](https://discord.gg/dWZkR6R)
[![Join the chat at https://playnet-ihtjamcsba.now.sh/](https://img.shields.io/badge/style-register-green.svg?style=social&label=Slack)](https://playnet-ihtjamcsba.now.sh/)


## Features

Implemented:
* Stable and Secure Rcon Connection
* Allow Management of Rcon Servers
	* Automated Server Restarts
	* Sending timed Messages/Commands to Servers
	* Server WatchDog
	* Streaming in-game Chats and Events to Console
	* Sending Server Log to Files (on Linux)
  
Planned: 
* Various Interfaces (API, CLI)
* Allow Management of Rcon Servers
  * Executing Rcon Commands
  * Offline Whitelisting
  * Provide in-game Chats to Interfaces
	* Provide Server Performance and Host Information to Interfaces
* Offering ease of use with exisiting Tools
* In addition to that, it is planned to make this Project extensible with Plugins

## Usage

The Tool consists of several parts.
Main Part is the RCon library which connects to a given server and sends commands/handles responses.
The other parts yet integrated are the Watcher and Scheduler.

### The Watcher
The Watcher is responsible for starting and watching your game process. When using the Watcher you always have to let the Tool start your ArmA server otherwise the process won't be detected.
When using the watcher as process manager, ending or killing gorcon-arma will also terminate your server process. This is a wanted feature as an automated restart of gorcon-arma would start a new server anyways which would then concur with the old one. On Windows the server is not being killed when gorcon-arma ends unexpected so take care of this when restarting it.

### The Scheduler
The Scheduler is able to either send a string over RCon (like: say -1 hello all) or send a restart command. If the Watcher is enabled, the restart will be done by sending a SIGTERM/SIGKILL to the process. If there is only RCon the restart will send a '#restartserver' command. Please note that without any kind of watcher your server might not come back up. When declaring a command in your config as restart event, the command string will be ignored.

Note that the Scheduler has it's own schedule.json file containing the timetable (see below).


### Getting Started

#### Binary
If you get the latest binary version from our [storage server](https://storage.play-net.org/minio/gorcon-arma/) you also get this README, the example config.json and an example schedule.json (HINT: You can select all the files you need and download them as a zip file). 

#### Debian Package (WIP)
If you want to use the pre-built debian package which also contains a systemd script for managing gorcon-arma (still being tested), you can get it by adding our bintray repository to your sources:
```
echo "deb https://dl.bintray.com/playnet/debian /" | sudo tee -a /etc/apt/sources.list
```
Then install via it's package name:
```
apt install gorcon-arma
```

#### Configuration
Once you got all required files installed you are ready to change the config.json according to your needs.
When entering the ArmA path make sure to use the right formating, even on Windows the path has to use forward slashes(/)! Do not escape spaces on Windows as Golang is handling all that for you.
Also note that logToFile and logToConsole are not supported on Windows, so we recommend keeping them disabled.
All further configuration options are described below. For both ```keepAliveTimer``` and ```keepAliveTolerance``` we recommend leaving them to the standards unless issues arise.

#### Starting
As gorcon-arma is a single binary starting it is fairly easy.
If you used the binary files simply start the gorcon-arma binary of your choice ```./gorcon-arma_linux-amd64```
If you used the Debian Package it is as simple as ```systemctl start gorcon-arma```

#### Debugging
If you encounter any issues with GoRcon-ArmA and need help, we recommend to first start with more output logging:
``` ./gorcon-arma_linux-amd64 --logtostderr=true -v=2```

The Verbosity Level is categorized in the following order:

- ```1``` Usual Output (can be always on)
- ```2``` More Info
- ```3``` Debug Communication
- ```4``` Debug Internals
- ```5``` Intense Debug
- ```10``` Loop Debugging

To give us feedback on your problems or to tell us about requests/ideas feel free to post them in our Issues Section on [Gitlab](https://git.play-net.org/playnet-public/gorcon-arma/issues) or [Github](https://github.com/playnet-public/gorcon-arma/issues)
We also happily invite you to join us on [Slack](https://playnet-ihtjamcsba.now.sh/) or [Discord](https://discord.gg/dWZkR6R)!

### Config Manual

```json
{
    "arma": {
        "enabled": true,
        "ip": "127.0.0.1",
        "port": "2301", 
        "password": "qwerty", 
        "keepAliveTimer": 10, 
        "keepAliveTolerance": 4,
        "showChat": true,
        "showEvents": true
    },

    "scheduler": {
        "enabled": true,
        "path": "schedule.json"
    },

    "watcher": {
        "enabled": true,
        "path": "D:/Program Files (x86)/Steam/SteamApps/common/Arma 3/arma3server.exe", 
        "params": [
            "-name=goTest",
            "-port-2303"
        ],
        "logToFile": true,
        "logFolder": "logs",
        "logToConsole": false
    }
}
```

**Explanation for ```arma``` section**
- ```enabled``` Whether or not RCon is enabled
- ```ip``` IP of the RCon Server
- ```port``` RCon Port as set in _beserver.cfg_
- ```password``` RCon Password as set in _beserver.cfg_
- ```keepAliveTimer``` The amount of seconds to wait until a keepAlivePacket is send to RCon (BattlEye Specification is min. 45sec)
- ```keepAliveTolerance``` The maximum tolerance between the sent keepAlives and the server response (higher means slower detection of disconnect, lower might cause unrequired reconnects)
- ```showChat``` Whether or not the Server Chat should be streamed to the console/stdout
- ```showEvents```Whether or not the Server Events should be streamed to the console/stdout

**Explanation for ```scheduler``` section**
- ```enabled``` Whether or not the scheduler is enabled
- ```path``` Path to schedule.json (keep local if not required otherwise)

**Explanation for ```watcher``` section**
- ```enabled``` Whether or not the watcher is enabled
- ```path``` Path to the ArmA executable (linux or windows)
- ```params``` Array of parameters for ArmA (watch formating for linux)
- ```logToFile``` Enable or Disable stderr/stdout logging of game server (linux systems only)
- ```logFolder``` Set the folder path in which logfiles are being created
- ```logToConsole``` Enables streaming of the server output(logs) to the console (linux systems only)

### Schedule Manual
The Scheduler implements a system like cronjobs. To learn more about it check out this [link](https://crontab.guru)

```json
{
    "schedule": [
        {
            "command": "say -1 Message every 5 minutes",
            "restart": false,
            "day": "*",
            "hour": "*",
            "minute": "*/5"
        }
    ]
}
```

- ```command``` Command to be executed (if not restart)
- ```restart``` If the Server should be restarted (overrides command)
- ```day``` Day of the Week to run the Event (0-6, 0 = Sunnday, * = Every Day)
- ```hour``` Hour of the Day to run the Event (0-23, * = Every Hour)
- ```minute``` Minute of the Hour to run the Event (0-60, * = Every Minute)

### Scheduler Examples

Example Event to restart the Server every hour at xx:30am/pm

```json
{
    "command": "",
    "restart": true,
    "day": "*",
    "hour": "*",
    "minute": "30"
}
```

Example Event to restart the Server every day at 12:08am

```json
{
    "command": "",
    "restart": true,
    "day": "*",
    "hour": "12",
    "minute": "8"
}
```

## Development

This project is using a [basic template](github.com/playnet-public/gocmd-template) for developing PlayNet command-line tools. Refer to this template for further information and usage docs.
The Makefile is configurable to some extent by providing variables at the top.
Any further changes should be thought of carefully as they might brake CI/CD compatibility.

One project might contain multiple tools whose main packages reside under `cmd`. Other packages like libraries go into the `pkg` directory.
Single projects can be handled by calling `make toolname maketarget` like for example:
```
make template dev
```
All tools at once can be handled by calling `make full maketarget` like for example:
```
make full build
```
Build output is being sent to `./build/`.

If you only package one tool this might seam slightly redundant but this is meant to provide consistence over all projects.
To simplify this, you can simply call `make maketarget` when only one tool is located beneath `cmd`. If there are more than one, this won't do anything (including not return 1) so be careful.

## Dependencies
This project has a pretty complex Makefile and therefore requires `make`.

Go Version: 1.8

Install all further requirements by running `make deps`

## Contributions

Pull Requests and Issue Reports are welcome.
If you are interested in contributing, feel free to [get in touch](https://discord.gg/WbrXWJB)


## License
This project is licensed under the included License (GNU GPLv3).
We also ask you to keep the projects name and links as they are, to direct possible contributors and users to the original sources.
Do not host releases yourself. Always redirect users back to the official releases for downloads.

Powered by https://play-net.org.