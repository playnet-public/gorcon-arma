package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime"
	"time"

	rcon "play-net.org/bercon"
	"play-net.org/gorcon-arma/procwatch"

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
	logToFile := cfg.GetBool("scheduler.logToFile")
	logFolder := cfg.GetString("scheduler.logFolder")
	logToConsole := cfg.GetBool("scheduler.logToConsole")
	useRcon := true
	showChat := cfg.GetBool("arma.showChat")
	showEvents := cfg.GetBool("arma.showEvents")

	var err error
	var watcher *procwatch.Watcher
	var client *rcon.Client
	var cmdChan chan string
	var stdout *io.ReadCloser
	var stderr *io.ReadCloser
	consoleOut, consoleIn := io.Pipe()

	// TODO: Refactor so scheduler and watcher are enabled separately
	if useSched {
		glog.Infof("Scheduler is enabled")
		watcher, err = runWatcher()
		if err != nil {
			return err
		}
		cmdChan = watcher.GetCmdChannel()
		stderr, stdout = watcher.GetOutput()
		if logToFile {
			go runFileLogger(stdout, stderr, logFolder)
		}
		if logToConsole {
			go runConsoleLogger(stdout, stderr, consoleIn)
		}
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
			go pipeCommands(cmdChan, client, consoleIn)
		}
		if showChat {
			client.SetChatWriter(consoleIn)
		}
		if showEvents {
			client.SetEventWriter(consoleIn)
		}
	} else {
		glog.Infof("RCon is disabled")
	}

	consoleScanner := bufio.NewScanner(consoleOut)
	for {
		for consoleScanner.Scan() {
			fmt.Printf(consoleScanner.Text())
		}
		if err := consoleScanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "There was an error with the consoleScanner", err)
		}
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

func runFileLogger(stdout, stderr *io.ReadCloser, logFolder string) {
	t := time.Now()
	logFileName := fmt.Sprintf("server_log_%v-%d-%v_%v-%v-%v.log", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute(), t.Second())
	logFile, err := os.Create(path.Join(logFolder, logFileName))
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	writer := bufio.NewWriter(logFile)
	defer writer.Flush()
	go io.Copy(writer, *stdout)
	go io.Copy(writer, *stderr)
}

func runConsoleLogger(stdout, stderr *io.ReadCloser, console io.Writer) {
	std := io.MultiReader(*stderr, *stdout)
	go io.Copy(console, std)
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
