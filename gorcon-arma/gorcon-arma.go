package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"runtime"

	"play-net.org/gorcon-arma/procwatch"
	"play-net.org/gorcon-arma/rcon"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	parameterMaxprocs = "maxprocs"
)

var (
	maxprocsPtr = flag.Int(parameterMaxprocs, runtime.NumCPU(), "max go procs")
)

var cfg *viper.Viper

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	flag.Parse()
	glog.Infoln("-- PlayNet GoRcon-ArmA - OpenSource Server Manager --")
	glog.Infoln("Version: 0.1.4")
	glog.Infoln("SourceCode: http://bit.ly/gorcon-code")
	glog.Infoln("Tasks: http://bit.ly/gorcon-tasks")
	glog.Infoln("")
	glog.Infoln("This project is work in progress - Use at your own risk")
	glog.Infoln("--")
	glog.Infof("Using %d go procs", *maxprocsPtr)
	runtime.GOMAXPROCS(*maxprocsPtr)

	if err := do(); err != nil {
		glog.Exit(err)
	}
}

func do() error {
	cfg = getConfig()
	useSched := cfg.GetBool("scheduler.enabled")
	useRcon := true

	var err error
	var watcher *procwatch.Watcher
	var client *rcon.Client
	var cmdChan chan string

	// TODO: Refactor so scheduler and watcher are enabled seperately
	if useSched {
		glog.Infof("Scheduler is enabled")
		watcher, err = runWatcher()
		if err != nil {
			return err
		}
		cmdChan = watcher.GetCmdChannel()
	} else {
		glog.Info("Scheduler is disabled")
	}

	if useRcon {
		glog.Infof("RCon is enabled")
		client, err = runRcon()
		if err != nil {
			return err
		}
		client.RunCommand("say -1 PlayNet GoRcon-ArmA Connected", nil)
		if useSched {
			go pipeCommands(cmdChan, client, nil)
		}
	} else {
		glog.Infof("RCon is disabled")
	}

	for {
	}
}

func runWatcher() (*procwatch.Watcher, error) {
	schedulerPath := procwatch.SchedulePath(cfg.GetString("scheduler.path"))
	schedulerEntity, err := schedulerPath.Parse()
	if err != nil {
		return nil, err
	}
	armaPath := cfg.GetString("arma.path")
	armaParam := cfg.GetString("arma.param")
	glog.Infof("\nScheduler Config: \n"+
		"Path to scheduler.json: %v \n"+
		"Path to ArmA Executable: %v \n"+
		"ArmA Parameters: %v \n",
		schedulerPath, armaPath, armaParam)
	pwcfg := procwatch.Cfg{
		A3exe:    armaPath,
		A3par:    armaParam,
		Schedule: *schedulerEntity,
	}

	watcher := procwatch.New(pwcfg)
	watcher.Start()
	return watcher, nil
}

func runRcon() (*rcon.Client, error) {
	armaIP := cfg.GetString("arma.ip")
	armaPort := cfg.GetString("arma.port")
	armaPassword := cfg.GetString("arma.password")
	armaKeepAliveTimer := cfg.GetInt("arma.keepAliveTimer")
	armaKeepAliveTolerance := cfg.GetInt64("arma.keepAliveTolerance")
	udpadr, err := net.ResolveUDPAddr("udp", armaIP+":"+armaPort)
	if err != nil {
		glog.Errorln("Could not convert ArmA IP and Port")
		return nil, err
	}
	glog.Infof("\nRCon Config: \n"+
		"ArmA Server Address: %v \n"+
		"ArmA Server Port: %v \n"+
		"KeepAliveTimer: %v \n"+
		"KeepAliveTolerance: %v",
		armaIP, armaPort, armaKeepAliveTimer, armaKeepAliveTolerance)
	becfg := rcon.Config{
		Addr:               udpadr,
		Password:           armaPassword,
		KeepAliveTimer:     armaKeepAliveTimer,
		KeepAliveTolerance: armaKeepAliveTolerance,
	}

	client := rcon.New(becfg)
	client.WatcherLoop()
	return client, nil
}

func pipeCommands(cmdChan chan string, c *rcon.Client, w io.WriteCloser) {
	for {
		cmd := <-cmdChan
		if len(cmd) != 0 {
			//TODO: Evaluate if this is good
			w.Write([]byte("Running Command: " + cmd))
			c.RunCommand(cmd, w)
		}
	}
}

func getConfig() *viper.Viper {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.AddConfigPath(".")

	glog.V(1).Infof("Reading Config")

	err := cfg.ReadInConfig()
	if err != nil {
		message := fmt.Sprintf("Loading Config failed with Error: %v", err.Error())
		glog.Errorln(message)
		panic(message)
	}
	return cfg
}
