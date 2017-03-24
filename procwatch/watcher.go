package procwatch

import (
	"io"
	"os/exec"
	"path"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/robfig/cron"
)

//Cfg contains all data required by Procwatch
type Cfg struct {
	A3exe    string
	A3par    string
	Schedule Schedule
	//Timezone int
}

//Config is the Interface providing Configs for the Procwatch
type Config interface {
	GetConfig() Cfg
}

//GetConfig returns the Cfg Object
func (c Cfg) GetConfig() Cfg {
	return c
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
	cmdChan   chan string
	stdout    io.ReadCloser
	stderr    io.ReadCloser
}

//New creates a Procwatch with given Config
func New(w Config) *Watcher {
	cfg := w.GetConfig()

	return &Watcher{
		a3exe:    cfg.A3exe,
		a3par:    cfg.A3par,
		schedule: cfg.Schedule,
		cron:     *cron.New(),
		cmdChan:  make(chan string),
	}
}

//Start the Server
func (w *Watcher) Start() {
	var err error
	w.cmd = exec.Command(w.a3exe, w.a3par)
	w.cmd.Dir = path.Dir(w.a3exe)
	w.stdout, err = w.cmd.StdoutPipe()
	if err != nil {
		glog.Error(err)
	}
	w.stderr, err = w.cmd.StderrPipe()
	if err != nil {
		glog.Error(err)
	}
	if err := w.cmd.Start(); err == nil {
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

//GetCmdChannel returns the channel to which scheduler and watcher write their commands
func (w *Watcher) GetCmdChannel() chan string {
	if w.cmdChan != nil {
		return w.cmdChan
	}
	return nil
}

//GetOutput returns the Stderr and Stdout Readers
func (w *Watcher) GetOutput() (stderr, stdout *io.ReadCloser) {
	stderr = &w.stderr
	stdout = &w.stdout
	return
}

//Wait for Server to exit
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
