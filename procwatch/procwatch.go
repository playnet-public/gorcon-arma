package procwatch

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"path"
	"sync"
	"time"

	"fmt"

	"github.com/golang/glog"
	"github.com/robfig/cron"
)

type SchedulerPath string

//Config contains all data required by Procwatch
type Config struct {
	A3exe    string
	A3par    string
	Schedule Schedule
	Timezone int
}

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

func (sc SchedulerPath) Parse() (*Schedule, error) {
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

//GetConfig retunrs WatcherConfig
func (wat Config) GetConfig() Config {
	return wat
}

//WatcherCfg is the Interface providing Configs for the Procwatch
type WatcherCfg interface {
	GetConfig() Config
}

//Watcher is the the Object Handling the Procwatch
type Watcher struct {
	a3exe     string
	a3par     string
	pid       uint32
	waitGroup sync.WaitGroup
	cmd       *exec.Cmd
	schedule  Schedule
	cron      cron.Cron
}

//New creates a Procwatch with given Config
func New(wat WatcherCfg) *Watcher {
	cfg := wat.GetConfig()

	return &Watcher{
		a3exe:    cfg.A3exe,
		a3par:    cfg.A3par,
		schedule: cfg.Schedule,
		cron:     *cron.New(),
	}
}

//Start the Server
func (w *Watcher) Start() {
	w.cmd = exec.Command(w.a3exe, w.a3par)
	w.cmd.Dir = path.Dir(w.a3exe)
	err := w.cmd.Start()
	if err == nil {
		w.pid = uint32(w.cmd.Process.Pid)
		w.waitGroup = sync.WaitGroup{}
		w.waitGroup.Add(1)
		go w.wait()
		err = w.buildJobs()
		if err != nil {
			glog.Error(err)
		}
	} else {
		return
	}
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
				glog.Info("Theoretischer Neustart per Scheduler")
			})
			if err != nil {
				return err
			}
		} else {
			err := w.cron.AddFunc(fmt.Sprintf("0 %s %s * * %s", minute, hour, day), func() {
				glog.Info(command)
			})
			if err != nil {
				return err
			}
		}

	}
	w.cron.Start()
	return nil
}

//Wait for Server exit
func (w *Watcher) wait() {
	defer w.waitGroup.Done()

	procwait, err := w.cmd.Process.Wait()
	if err != nil {
		return
	}

	if procwait.Exited() {
		w.restart()
	}
}

//Restart the Server
func (w *Watcher) restart() {
	time.Sleep(time.Second * 5)
	w.pid = 0
	w.Start()
}
