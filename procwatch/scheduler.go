package procwatch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"syscall"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
)

//SchedulePath to config
type SchedulePath string

//Schedule Object
type Schedule struct {
	Schedule []SchedulerEntity `json:"schedule"`
}

//SchedulerEntity all data required by Procwatch
type SchedulerEntity struct {
	Command string `json:"command"`
	Restart bool   `json:"restart"`
	Day     string `json:"day"`
	Hour    string `json:"hour"`
	Minute  string `json:"minute"`
}

//Parse json from path and return Schedule
func (sc SchedulePath) Parse() (*Schedule, error) {
	content, err := ioutil.ReadFile(string(sc))
	if err != nil {
		return nil, err
	}
	return parseConfig(content)
}

func parseConfig(content []byte) (*Schedule, error) {
	config := &Schedule{}
	if err := json.Unmarshal(content, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (w *Watcher) buildJobs() (err error) {
	defer func() {
		if err != nil {
			raven.CaptureErrorAndWait(err, map[string]string{"app": "procwatch.scheduler"})
		}
	}()
	scheduleArr := w.schedule.Schedule
	glog.V(1).Infoln("Scheduling Commands: ")
	for index := 0; index < len(scheduleArr); index++ {
		scheduleEntry := scheduleArr[index]
		command := scheduleEntry.Command
		restart := scheduleEntry.Restart
		day := scheduleEntry.Day
		hour := scheduleEntry.Hour
		minute := scheduleEntry.Minute
		glog.V(1).Infof("Adding Event at %s %s * * %s", minute, hour, day)
		if restart {
			err := w.cron.AddFunc(fmt.Sprintf("0 %s %s * * %s", minute, hour, day), func() {
				if w.useWatcher {
					glog.V(2).Infoln("Sending Termination Signal to Process")
					err := w.cmd.Process.Signal(syscall.SIGTERM)
					if err != nil {
						if err.Error() != "not supported by windows" {
							glog.Error(err)
						}
						err := w.cmd.Process.Signal(syscall.SIGKILL)
						if err != nil {
							glog.Error(err)
						}
					}
				} else {
					glog.V(2).Infoln("Sending Restart Command to Channel")
					w.cmdChan <- "#restartserver"
				}
			})
			if err != nil {
				return err
			}
		} else {
			err := w.cron.AddFunc(fmt.Sprintf("0 %s %s * * %s", minute, hour, day), func() {
				glog.V(2).Infoln("Sending Command to Channel: ", command)
				w.cmdChan <- command
			})
			if err != nil {
				return err
			}
		}
	}
	w.cron.Start()
	return nil
}
