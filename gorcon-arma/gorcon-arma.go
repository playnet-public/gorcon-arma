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

	rcon "git.play-net.org/playnet-public/bercon"
	"git.play-net.org/playnet-public/gorcon-arma/procwatch"

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
	fmt.Println("-- PlayNet GoRcon-ArmA - OpenSource Server Manager --")
	fmt.Println("Version: 0.1.4")
	fmt.Println("SourceCode: http://bit.ly/gorcon-code")
	fmt.Println("Tasks: http://bit.ly/gorcon-tasks")
	fmt.Println("")
	fmt.Println("This project is work in progress - Use at your own risk")
	fmt.Println("--")
	fmt.Println("")
	fmt.Printf("Using %d go procs\n", *maxprocsPtr)
	runtime.GOMAXPROCS(*maxprocsPtr)

	if err := do(); err != nil {
		glog.Fatal(err)
	}
}

func do() error {
	cfg = getConfig()
	useSched := cfg.GetBool("scheduler.enabled")
	useWatch := cfg.GetBool("watcher.enabled")
	useRcon := cfg.GetBool("arma.enabled")
	logToFile := cfg.GetBool("watcher.logToFile")
	logFolder := cfg.GetString("watcher.logFolder")
	logToConsole := cfg.GetBool("watcher.logToConsole")
	showChat := cfg.GetBool("arma.showChat")
	showEvents := cfg.GetBool("arma.showEvents")

	quit := make(chan int)

	var err error
	var watcher *procwatch.Watcher
	var client *rcon.Client
	var cmdChan chan string
	var stdout io.ReadCloser
	var stderr io.ReadCloser
	consoleOut, consoleIn := io.Pipe()
	go streamConsole(consoleOut)
	// TODO: Refactor so scheduler and watcher are enabled separately
	if useSched || useWatch {
		glog.V(4).Infoln("Starting Procwatch")
		watcher, err = runWatcher(useSched, useWatch)
		if err != nil {
			return err
		}
		glog.V(4).Infoln("Retrieving Procwatch Command Channel")
		cmdChan = watcher.GetCmdChannel()
		glog.V(4).Infoln("Retrieving Procwatch Output Channels")
		stderr, stdout = watcher.GetOutput()
		if logToFile && useWatch {
			go runFileLogger(stdout, stderr, logFolder)
		}
		if logToConsole && useWatch {
			go runConsoleLogger(stdout, stderr, consoleIn)
		}
	} else {
		fmt.Println("Scheduler is disabled")
	}

	if useRcon {
		fmt.Println("RCon is enabled")
		client, err = runRcon()
		if err != nil {
			return err
		}
		if useSched {
			go pipeCommands(cmdChan, client, nil)
		}
		if showChat {
			client.SetChatWriter(consoleIn)
		}
		if showEvents {
			client.SetEventWriter(consoleIn)
		}
		client.RunCommand("say -1 PlayNet GoRcon-ArmA Connected", nil)
	} else {
		fmt.Println("RCon is disabled")
	}

	q := <-quit
	if q == 1 {
		return nil
	}
	return nil
}

func runWatcher(useSched, useWatch bool) (watcher *procwatch.Watcher, err error) {
	var armaPath string
	var armaParam string
	var schedulerEntity *procwatch.Schedule

	if useSched {
		schedulerPath := procwatch.SchedulePath(cfg.GetString("scheduler.path"))
		schedulerEntity, err = schedulerPath.Parse()
		if err != nil {
			return
		}
		fmt.Println("\nScheduler is enabled")
		fmt.Printf("\nScheduler Config: \n"+
			"Path to scheduler.json: %v \n",
			schedulerPath)
	} else {
		schedulerEntity = &procwatch.Schedule{}
	}

	if useWatch {
		armaPath = cfg.GetString("watcher.path")
		armaParam = cfg.GetString("watcher.params")
		fmt.Println("\nWatcher is enabled")
		fmt.Printf("\nWatcher Config: \n"+
			"Path to ArmA Executable: %v \n"+
			"ArmA Parameters: %v \n\n",
			armaPath, armaParam)
	}

	pwcfg := procwatch.Cfg{
		A3exe:        armaPath,
		A3par:        armaParam,
		Schedule:     *schedulerEntity,
		UseScheduler: useSched,
		UseWatcher:   useWatch,
	}

	watcher = procwatch.New(pwcfg)
	watcher.Start()
	return
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
	fmt.Printf("\nRCon Config: \n"+
		"ArmA Server Address: %v \n"+
		"ArmA Server Port: %v \n"+
		"KeepAliveTimer: %v \n"+
		"KeepAliveTolerance: %v \n\n",
		armaIP, armaPort, armaKeepAliveTimer, armaKeepAliveTolerance)
	becfg := rcon.Config{
		Addr:               udpadr,
		Password:           armaPassword,
		KeepAliveTimer:     armaKeepAliveTimer,
		KeepAliveTolerance: armaKeepAliveTolerance,
	}

	client := rcon.New(becfg)
	fmt.Println("Establishing Connection to Server")
	client.WatcherLoop()
	return client, nil
}

func runFileLogger(stdout, stderr io.ReadCloser, logFolder string) {
	t := time.Now()
	logFileName := fmt.Sprintf("server_log_%v%d%v_%v-%v-%v.log", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	fmt.Println("Creating Server Logfile: ", logFileName)
	logFile, err := os.Create(path.Join(logFolder, logFileName))
	if err != nil {
		panic(err)
	}
	defer logFile.Close()

	writer := bufio.NewWriter(logFile)
	defer writer.Flush()
	go io.Copy(writer, stdout)
	go io.Copy(writer, stderr)
}

func runConsoleLogger(stdout, stderr io.ReadCloser, console io.Writer) {
	std := io.MultiReader(stderr, stdout)
	go io.Copy(console, std)
}

func streamConsole(consoleOut io.Reader) error {
	consoleScanner := bufio.NewScanner(consoleOut)
	for consoleScanner.Scan() {
		fmt.Println(consoleScanner.Text())
	}
	if err := consoleScanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "There was an error with the consoleScanner", err)
		return err
	}
	return nil
}

func pipeCommands(cmdChan chan string, c *rcon.Client, w io.WriteCloser) {
	for {
		glog.V(10).Infoln("Looping pipeCommands")
		cmd := <-cmdChan
		if len(cmd) != 0 {
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
