package rcon

import "github.com/playnet-public/gorcon-arma/common"
import "os"

//InjectExtFuncs takes a map of functions and adds Watcher Functions
func (c *Client) InjectExtFuncs(funcs common.ScheduleFuncs) common.ScheduleFuncs {
	funcs["rcon"] = c.extFuncs()
	return funcs
}

func (c *Client) extFuncs() common.ScheduleFunc {
	return func(cmd string) {
		//TODO: Maybe add a better way to output the result
		c.Exec([]byte(cmd), os.Stdout)
	}
}
