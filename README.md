# GoRcon-ArmA
The Go based Rcon solution for ArmA servers with server management features

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
        "keepAliveTolerance": 4 
    },
    "scheduler": {
        // Wheteher or not the scheduler is enabled
        "enabled": true,
        // Path to schedule.json (keep local if not required otherwise)
        "path": "schedule.json",
        // The Timezone Offset from GMT (Berlin: 1)
        "timezone": 1 
    }
}
```

## License
This project is licensed under the included License (GNU GPLv3).
Powered by https://play-net.org
