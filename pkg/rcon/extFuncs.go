package rcon

import (
	"os"

	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/pkg/common"
)

//ExtFuncs returns functions to be externally exposed
func (c *Client) ExtFuncs() common.ExtFuncs {
	f := common.NewExtFunc("rcon", c.extFuncs())
	return common.NewExtFuncs(f)
}

func (c *Client) extFuncs() common.ScheduleFunc {
	return func(cmd string) {
		//TODO: Maybe add a better way to output the result
		c.Exec([]byte(cmd), os.Stdout)
	}
}

//ExtFuncs returns functions to be externally exposed
func (em *EventManager) ExtFuncs() common.ExtFuncs {
	f := common.NewExtFunc("events", em.dbgFuncs())
	return common.NewExtFuncs(f)
}

func (em *EventManager) dbgFuncs() common.ScheduleFunc {
	return func(cmd string) {
		switch cmd {
		case "get":
			glog.Infoln("All Events:", em.Get())
		}
	}
}

//ExtFuncs returns functions to be externally exposed
func (mm *MessageManager) ExtFuncs() common.ExtFuncs {
	f := common.NewExtFunc("messages", mm.dbgFuncs())
	return common.NewExtFuncs(f)
}

func (mm *MessageManager) dbgFuncs() common.ScheduleFunc {
	return func(cmd string) {
		switch cmd {
		case "get":
			glog.Infoln("All Messages:", mm.Get())
		}
	}
}
