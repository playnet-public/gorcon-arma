package watcher

import "github.com/playnet-public/gorcon-arma/pkg/common"
import "github.com/golang/glog"

//ExtFuncs returns functions to be externally exposed
func (w *Watcher) ExtFuncs() common.ExtFuncs {
	f := common.NewExtFunc("watcher", w.extFuncs())
	return common.NewExtFuncs(f)
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
