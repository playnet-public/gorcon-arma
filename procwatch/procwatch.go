package procwatch

import (
	"os/exec"
	"path"
	"sync"
	"time"
)

//Config contains all data required by Procwatch
type Config struct {
	A3exe    string
	A3par    string
	Schedule []SchedulerEntity
}

//SchedulerEntity all data required by Procwatch
type SchedulerEntity struct {
	Command string `json:"command"`
	Restart bool   `json:"restart"`
	Day     string `json:"day"`
	Hour    string `json:"hour"`
	Minute  string `json:"minute"`
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
}

//New creates a Procwatch with given Config
func New(wat WatcherCfg) *Watcher {
	cfg := wat.GetConfig()

	return &Watcher{
		a3exe: cfg.A3exe,
		a3par: cfg.A3par,
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
	} else {
		return
	}
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
