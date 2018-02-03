package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/robfig/cron"
)

//Schedule Object
type Schedule struct {
	Events []Event `json:"schedule"`
}

//Event all data required by Procwatch
type Event struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Day     string `json:"day"`
	Hour    string `json:"hour"`
	Minute  string `json:"minute"`
}

//Scheduler executes functions based on a schedule
type Scheduler struct {
	Funcs common.ScheduleFuncs
	Sched Schedule
	cron  *cron.Cron
}

//New returns a new Scheduler instance
func New(
	sched Schedule,
	funcs common.ScheduleFuncs,
) *Scheduler {
	cron := cron.New()
	scheduler := &Scheduler{
		Sched: sched,
		Funcs: funcs,
		cron:  cron,
	}
	scheduler.Funcs["scheduler"] = scheduler.scheduleFunc
	return scheduler
}

func (s *Scheduler) scheduleFunc(cmd string) {
	if cmd == "" {
		glog.Errorln("no cmd in scheduleFunc call")
		return
	}
	cmds := strings.Split(cmd, " ")
	switch cmds[0] {
	case "reload":
		if len(cmds) > 1 {
			glog.Warningln("no path provided for schedule.reload")
			s.Reload(cmds[1])
		} else {
			s.Reload("")
		}

	case "stop":
		s.Stop()
	}
}

//BuildEvents adds all time events to the Scheduler's cron
func (s *Scheduler) BuildEvents() (err error) {
	defer func() {
		if err != nil {
			raven.CaptureErrorAndWait(err, map[string]string{"app": "procwatch.scheduler"})
		}
	}()
	events := s.Sched.Events
	glog.V(2).Infoln("Scheduling Events: ")
	for index := 0; index < len(events); index++ {
		event := events[index]
		eventType := event.Type
		command := event.Command
		day := event.Day
		hour := event.Hour
		minute := event.Minute
		glog.V(2).Infof("Adding Event at %s %s * * %s", minute, hour, day)

		eventFunc, ok := s.Funcs[eventType]
		if ok {
			err = s.cron.AddFunc(fmt.Sprintf("0 %s %s * * %s", minute, hour, day), func() {
				eventFunc(command)
			})
			if err != nil {
				return
			}
		} else {
			err = errors.New("no function defined for eventType " + eventType)
			glog.Errorln(err)
			return
		}
	}
	return
}

//Start the cron loop
func (s *Scheduler) Start() {
	glog.V(1).Infoln("Starting Scheduler Jobs")
	s.cron.Start()
}

//Stop the cron loop
func (s *Scheduler) Stop() {
	glog.V(1).Infoln("Stopping Scheduler Jobs")
	s.cron.Stop()
}

//Reload the scheduler jobs from file
func (s *Scheduler) Reload(path string) (err error) {
	s.Stop()
	s.cron = cron.New()
	s.Sched, err = ReadSchedule(path)
	if err != nil {
		glog.Errorln(err)
		return
	}
	err = s.BuildEvents()
	if err != nil {
		glog.Errorln(err)
		return
	}
	s.Start()
	return
}

//UpdateFuncs recreates the common.ScheduleFuncs from funcs
func (s *Scheduler) UpdateFuncs(funcs ...common.ExtFuncs) {
	for _, extF := range funcs {
		for _, f := range extF {
			s.Funcs[f.Key] = f.Func
		}
	}
}

//ReadSchedule json from path and return Schedule
func ReadSchedule(path string) (Schedule, error) {
	if path == "" {
		path = "schedule.json"
	}
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return Schedule{}, err
	}
	return parseConfig(content)
}

func parseConfig(content []byte) (Schedule, error) {
	config := &Schedule{}
	if err := json.Unmarshal(content, config); err != nil {
		return Schedule{}, err
	}
	return *config, nil
}
