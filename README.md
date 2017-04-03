# GoRcon-ArmA
The Go based Rcon solution for ArmA servers with server management features

Master: [![build status](https://git.play-net.org/playnet-public/gorcon-arma/badges/master/build.svg)](https://git.play-net.org/playnet-public/gorcon-arma/commits/master)
Develop: [![build status](https://git.play-net.org/playnet-public/gorcon-arma/badges/develop/build.svg)](https://git.play-net.org/playnet-public/gorcon-arma/commits/develop)


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
Main Part is the RCon library which connects to a given Server and sends commands/handles responses.
The other parts yet integrated are the Watcher and Scheduler.

### The Watcher
The Watcher is responsible for starting and watching your game process. When using the Watcher you always have to let the Tool start your ArmA Server otherwise the process won't be detected.

### The Scheduler
The Scheduler is able to either send a String over RCon (like: say -1 hello all) or send a restart command. If the Watcher is enabled, the restart will be done by sending a SIGTERM/SIGKILL to the Process. If there is only RCon the restart will send a '#restartserver' command. Please note that without any watcher your server might not come back up.
Note that the Scheduler has it's own schedule.json file containing the timetable (see below).


### Config Manual
```json
{
    "arma": {
        // Whether or not RCon is enabled
        "enabled": true,
        // IP of the RCon Server
        "ip": "127.0.0.1",
        // RCon Port as set in beserver.cfg
        "port": "2301", 
        // RCon Password as set in beserver.cfg
        "password": "qwerty", 
        // The amount of seconds to wait until a keepAlivePacket is send to RCon (BattlEye Specification is min. 45sec)
        "keepAliveTimer": 10, 
        // The maximum tolerance between the sent keepAlives and the Servers response (higher means slower detection of disconnect, lower might cause unrequired reconnects)
        "keepAliveTolerance": 4,
        // Whether or not the Server Chat should be streamed to the console/stdout
        "showChat": true,
        // Whether or not the Server Events should be streamed to the console/stdout
        "showEvents": true
    },

    "scheduler": {
        // Wheteher or not the scheduler is enabled
        "enabled": true,
        // Path to schedule.json (keep local if not required otherwise)
        "path": "schedule.json",
    },

    "watcher": {
        // Wheteher or not the watcher is enabled
        "enabled": true,
        // Path to the ArmA executable (linux or windows)
        "path": "D:/Program Files (x86)/Steam/SteamApps/common/Arma 3/arma3server.exe", 
        // single string of parameters for ArmA (watch formating for linux)
        "params": "-name=goTest",
        // Enable or Disable stderr/stdout logging of game server (useful on linux systems)
        "logToFile": true,
        // Set the folder path in which logfiles are being created
        "logFolder": "logs",
        // Enables streaming of the server output(logs) to the console (warning: might be very verbose)
        "logToConsole": false
    }
}
```

### Schedule Manual
The Scheduler implements a system like cronjobs. To learn more about it check out this [link](https://crontab.guru)
```json
{
    "schedule": [
        //One Schedule Event
        {
            // Command to be executed (if not restart)
            "command": "say -1 Restart in 30 minutes",
            // If the Server should be restarted (overrides command)
            "restart": false,
            // Day of the Week to run the Event (0-6, 0 = Sunnday, * = Every Day)
            "day": "*",
            // Hour of the Day to run the Event (0-23, * = Every Hour)
            "hour": "*",
            // Minute of the Hour to run the Event (0-60, * = Every Minute)
            "minute": "11"
        },
        
        // Example Event to restart the Server every day at 12:08am
        {
            "command": "",
            "restart": true,
            "day": "*",
            "hour": "12",
            "minute": "8"
        },

        // Example Event to restart the Server every hour at xx:30am/pm
        {
            "command": "",
            "restart": true,
            "day": "*",
            "hour": "*",
            "minute": "30"
        }
    ]
}
```

## License
This project is licensed under the included License (GNU GPLv3).
We also ask you to keep the projects name and links as they are, to direct possible contributors and users to the original sources.
Do not host releases yourself. Always redirect users back to the official releases for downloads.

Powered by https://play-net.org.
