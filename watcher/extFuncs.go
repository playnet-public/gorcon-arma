package watcher

import "github.com/playnet-public/gorcon-arma/common"
import "github.com/golang/glog"

//InjectExtFuncs takes a map of functions and adds Watcher Functions
func (w *Watcher) InjectExtFuncs(funcs common.ScheduleFuncs) common.ScheduleFuncs {
	funcs["watcher"] = w.extFuncs()
	return funcs
}

func (w *Watcher) extFuncs() common.ScheduleFunc {
	return func(cmd string) {
		switch cmd {
		case "restart":
			glog.Infoln("Triggering restart")
			w.BlockRestart = false
			w.Stop()
		case "stop":
			glog.Infoln("Triggering stop")
			w.BlockRestart = true
			w.Stop()
		case "start":
			glog.Infoln("Triggering start")
			w.BlockRestart = false
			w.Exec()
		case "watch":
			glog.Infoln("Triggering watch")
			w.BlockRestart = false
			w.Watch(w.RestartAndWatch)
		}
	}
}
