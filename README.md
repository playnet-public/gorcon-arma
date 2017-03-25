# GoRcon-ArmA
The Go based Rcon solution for ArmA servers with server management features

Master: [![build status](https://git.play-net.org/playnet-public/gorcon-arma/badges/master/build.svg)](https://git.play-net.org/playnet-public/gorcon-arma/commits/master)
Develop: [![build status](https://git.play-net.org/playnet-public/gorcon-arma/badges/develop/build.svg)](https://git.play-net.org/playnet-public/gorcon-arma/commits/develop)


## Features

Planned: 
* Stable and Secure Rcon Connection with easy to use Interfaces
* Allow Management of Rcon Servers
  * Executing Rcon Commands
  * Automated Server Restarts
  * Sending timed Messages to Players
  * Offline Whitelisting
  * Providing in-game Chats to Interfaces
  * Server WatchDog
* Offering ease of use with exisiting Tools
* In addition to that, it is planned to make this Project extendible with Plugins

## Usage

The Tool consists of several parts.
Main Part is the rcon library which connects to a given Server and sends commands/handles responses.
The other part yet integrated is the scheduler. It allows to define time sets when commands should be executed.

The Scheduler is able to either send a String over RCon (like: say -1 hello all) or send a restart command.
For now both happens over rcon and therefor a working connection is required, but in the near future it is planned to also send the exit command to the process itself.

### Config Manual
```json
{
    "arma": {
        // IP of the RCon Server
        "ip": "127.0.0.1",
        // RCon Port as set in beserver.cfg
        "port": "2301", 
        // RCon Password as set in beserver.cfg
        "password": "qwerty", 
        // Path to the ArmA executable (linux or windows)
        "path": "D:/Program Files (x86)/Steam/SteamApps/common/Arma 3/arma3server.exe", 
        // single string of parameters for ArmA (watch formating for linux)
        "param": "-name=goTest",
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
            "restart": true,
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
Do not host releases yourself and redirect users back to the official releases for downloads.

Powered by https://play-net.org.
