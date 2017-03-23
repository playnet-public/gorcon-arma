package procwatch

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

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

func (w *Watcher) buildJobs() error {
	scheduleArr := w.schedule.Schedule
	for index := 0; index < len(scheduleArr); index++ {
		scheduleEntry := scheduleArr[index]
		command := scheduleEntry.Command
		restart := scheduleEntry.Restart
		day := scheduleEntry.Day
		hour := scheduleEntry.Hour
		minute := scheduleEntry.Minute
		glog.Info(fmt.Sprintf("%s %s * * %s", minute, hour, day))
		if restart {
			err := w.cron.AddFunc(fmt.Sprintf("0 %s %s * * %s", minute, hour, day), func() {
				glog.V(2).Infoln("Sending Restart Command to Channel")
				w.cmdChan <- "#restartserver"
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
