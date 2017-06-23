package watcher

import (
	"io"
	"os/exec"
	"runtime"
	"syscall"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
)

//Process describes the process to watch
type Process struct {
	ExecPath       string
	ExecDir        string
	ExecParam      []string
	StdErr, StdOut io.Writer
}

type restartFunc func()

//Watcher is the the Object Handling the Procwatch
type Watcher struct {
	Proc Process

	OnExit       restartFunc
	Delay        int
	BlockRestart bool
	cmd          *exec.Cmd
}

//New returns a Watcher for Process
func New(p Process) *Watcher {
	return &Watcher{
		Proc:  p,
		Delay: 5,
	}
}

//Exec runs the Process
func (w *Watcher) Exec() (err error) {
	glog.V(2).Infoln("Preparing Process")
	w.cmd = exec.Command(w.Proc.ExecPath, w.Proc.ExecParam...)
	w.cmd.Dir = w.Proc.ExecDir
	w.cmd.Stderr = w.Proc.StdErr
	w.cmd.Stdout = w.Proc.StdOut

	glog.V(1).Infoln("Executing Process")
	err = w.cmd.Start()
	if err != nil {
		raven.CaptureError(err, map[string]string{"app": "watcher"})
		glog.Fatalln(err)
		return
	}
	return
}

//Watch greates a go routine watching the process and it executes action on proc exit
func (w *Watcher) Watch(restartFunc restartFunc) {
	w.OnExit = restartFunc
	go w.wait()
}

//Wait for Server to exit
func (w *Watcher) wait() {
	glog.V(2).Infoln("Waiting for Process to Exit")
	_, err := w.cmd.Process.Wait()
	glog.Infoln("Process Exited - Error:", err)
	if err != nil {
		raven.CaptureError(err, map[string]string{"app": "watcher"})
		return
	}

	if w.BlockRestart {
		glog.Infoln("Restart is blocked by user. Exiting")
		return
	}

	//TODO: Check why or if this is required
	//if proc.Exited() {
	w.OnExit()
	//}
}

//Stop the Process and kill if necessary
func (w *Watcher) Stop() error {
	glog.V(2).Infoln("Sending Termination Signal to Process")
	if runtime.GOOS == "windows" {
		err := w.cmd.Process.Signal(syscall.SIGKILL)
		if err != nil {
			glog.Error(err)
			return err
		}
	} else {
		err := w.cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			glog.Error(err)
			err := w.cmd.Process.Signal(syscall.SIGKILL)
			if err != nil {
				glog.Error(err)
				return err
			}
		}
	}
	return nil
}

//Restart the Process with a short delay
func (w *Watcher) Restart() {
	time.Sleep(time.Second * time.Duration(w.Delay))
	glog.Infoln("Restarting Process")
	w.Exec()
}

//RestartAndWatch restarts the proc and keeps watching it
func (w *Watcher) RestartAndWatch() {
	w.Restart()
	go w.wait()
}

//GetStd directly returns the proc stderr and stdout
/*
func (w *Watcher) GetStd() (stderr, stdout io.ReadCloser, err error) {
	stderr, err = w.cmd.StderrPipe()
	if err != nil {
		raven.CaptureError(err, map[string]string{"app": "watcher"})
		glog.Errorln(err)
	}
	stdout, err = w.cmd.StdoutPipe()
	if err != nil {
		raven.CaptureError(err, map[string]string{"app": "watcher"})
		glog.Errorln(err)
	}
	return
}
*/
