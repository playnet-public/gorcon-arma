package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime"
	"time"

	raven "github.com/getsentry/raven-go"
	"github.com/kolide/kit/version"
	bercon "github.com/playnet-public/gorcon-arma/pkg/bercon/client"
	"github.com/playnet-public/gorcon-arma/pkg/bercon/funcs"
	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
	"github.com/playnet-public/gorcon-arma/pkg/scheduler"
	"github.com/playnet-public/gorcon-arma/pkg/watcher"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"strings"

	"os/exec"

	"github.com/golang/glog"
	"github.com/spf13/viper"
)

const (
	app                 = "PlayNet GoRcon-ArmA - OpenSource Server Manager"
	appKey              = "gorcon-arma"
	parameterMaxprocs   = "maxprocs"
	parameterConfigPath = "configPath"
	parameterDevBuild   = "devbuild"
)

var (
	maxprocsPtr   = flag.Int(parameterMaxprocs, runtime.NumCPU(), "max go procs")
	configPathPtr = flag.String(parameterConfigPath, ".", "config parent folder")
	devBuildPtr   = flag.Bool(parameterDevBuild, false, "set dev build mode")
	versionPtr    = flag.Bool("version", true, "show or hide version info")
	dbgPtr        = flag.Bool("debug", false, "debug printing")
)

var cfg *viper.Viper

func main() {
	flag.Parse()

	if *versionPtr {
		fmt.Printf("-- PlayNet %s --\n", app)
		version.PrintFull()
	}
	runtime.GOMAXPROCS(*maxprocsPtr)

	// prepare glog
	defer glog.Flush()
	glog.CopyStandardLogTo("info")

	var zapFields []zapcore.Field
	// hide app and version information when debugging
	if !*dbgPtr {
		zapFields = []zapcore.Field{
			zap.String("app", appKey),
			zap.String("version", version.Version().Version),
		}
	}

	// prepare zap logging
	log := newLogger(*dbgPtr).With(zapFields...)
	defer log.Sync()
	log.Info("preparing")

	raven.CapturePanicAndWait(func() {
		if err := do(); err != nil {
			glog.Fatal(err)
			raven.CaptureErrorAndWait(err, map[string]string{"isFinal": "true"})
		}
	}, nil)
}

func do() (err error) {
	cfg = getConfig()

	if !*devBuildPtr {
		raven.SetDSN(cfg.GetString("playnet.sentry"))
		raven.SetIncludePaths([]string{
			"github.com/playnet-public/gorcon-arma/pkg/",
		})
		raven.SetRelease(version.Version().Version)
	}

	useSched := cfg.GetBool("scheduler.enabled")
	useWatch := cfg.GetBool("watcher.enabled")
	logToConsole := cfg.GetBool("watcher.logToConsole")
	logToFile := cfg.GetBool("watcher.logToFile")
	logFolder := cfg.GetString("watcher.logFolder")
	useRcon := cfg.GetBool("arma.enabled")
	var sched *scheduler.Scheduler
	var watch *watcher.Watcher
	var client *rcon.Client

	quit := make(chan error)

	if useSched {
		sched, err = newScheduler()
		if err != nil {
			return
		}
	}

	var stderr, stdout io.Writer
	if logToFile {
		logFile := newLogfile(logFolder)
		stdout = io.MultiWriter(logFile)
		stderr = io.MultiWriter(logFile)
		defer logFile.Close()
	}
	if logToConsole {
		stderr = io.MultiWriter(stderr, os.Stderr)
		stdout = io.MultiWriter(stdout, os.Stdout)
	}

	if useWatch {
		watch, err = newProcWatch(stderr, stdout)
		if err != nil {
			return
		}
		if useSched {
			sched.UpdateFuncs(watch.ExtFuncs())
		}
	}

	if useRcon {
		rconQ := make(chan error, 100)
		client, err = newRcon(rconQ)
		if err != nil {
			return
		}

		rconFuncs := funcs.New(client)

		bm := newBanManager()
		bm.AddCheck(bm.CheckLocal)
		//bm.AddCheck(testCheck)

		eventReader, eventWriter := io.Pipe()
		pm := newPlayerManager(*rconFuncs, bm)

		pm.Listen(eventReader, quit)

		go func() {
			for e := range rconQ {
				//Once the RCon Client finishes a reconnect, we check all Players for Bans and clean them up
				if e == common.ErrConnected {
					pm.CheckPlayers()
				}
				glog.V(2).Infoln("RCon Channel Return:", e)
			}
		}()

		//Take care of the writers passed here!
		//Writers other than os.Std* will always be closed after execution
		client.Exec([]byte("say -1 PlayNet GoRcon-ArmA Connected"), os.Stdout)

		//client.AttachChat(io.MultiWriter(eventWriter, stdout))
		client.AttachEvents(io.MultiWriter(eventWriter, stdout))

		if useSched {
			sched.UpdateFuncs(
				client.ExtFuncs(),
			)
		}
	}

	//Finish Func and Event Collection and start Scheduling
	if useSched {
		sched.BuildEvents()
		sched.Start()
	}

	q := <-quit
	err = q
	return q
}

func testCheck(desc string) (status bool, ban *rcon.Ban) {
	//glog.Infoln("Running testCheck for", desc)
	switch desc {
	case "69feed9123832b560f7d8b073eebf477":
		ban = &rcon.Ban{
			Descriptor: "69feed9123832b560f7d8b073eebf477",
			Reason:     "Some external test Ban",
		}
		status = true
	default:
		ban = nil
		status = false
	}
	//glog.Infoln("Sleeping before ban return")
	time.Sleep(time.Second * 10)
	return status, ban
}

func newRcon(q chan error) (*rcon.Client, error) {
	beIP := cfg.GetString("arma.ip")
	bePort := cfg.GetString("arma.port")
	bePassword := cfg.GetString("arma.password")
	beKeepAliveTimer := cfg.GetInt("arma.keepAliveTimer")
	beKeepAliveTolerance := cfg.GetInt64("arma.keepAliveTolerance")

	beCred := rcon.Credentials{
		Username: "",
		Password: bePassword,
	}

	beConAddr, err := net.ResolveUDPAddr("udp", beIP+":"+bePort)
	if err != nil {
		return nil, err
	}

	beCon := rcon.Connection{
		Addr:               beConAddr,
		KeepAliveTimer:     beKeepAliveTimer,
		KeepAliveTolerance: beKeepAliveTolerance,
	}
	beCl := bercon.New(beCon, beCred)
	rc := rcon.NewClient(
		beCl.Loop,
		beCl.Disconnect,
		beCl.Exec,
		beCl.AttachEvents,
		beCl.AttachChat,
	)
	glog.Infoln("Establishing Connection to Server")
	err = rc.Connect(q)
	if err != nil {
		return nil, err
	}
	return rc, nil
}

func newBanManager() *rcon.BanManager {
	return rcon.NewBanManager()
}

func newPlayerManager(funcs rcon.Funcs, bm *rcon.BanManager) *rcon.PlayerManager {
	pm := rcon.NewPlayerManager(funcs, bm)
	return pm
}

func newProcWatch(stderr, stdout io.Writer) (w *watcher.Watcher, err error) {
	execPath := cfg.GetString("watcher.exec")
	execDir := cfg.GetString("watcher.dir")
	if execDir == "" {
		execDir = path.Dir(execPath)
	}
	execParam := cfg.GetStringSlice("watcher.params")

	proc := watcher.Process{
		ExecPath:  execPath,
		ExecDir:   execDir,
		ExecParam: execParam,
		StdErr:    stderr,
		StdOut:    stdout,
	}

	w = watcher.New(proc)
	err = w.Exec()
	if err != nil {
		return
	}
	if cfg.GetBool("watcher.autoRestart") {
		w.Watch(w.RestartAndWatch)
	}
	return w, nil
}

func logCmd(cmd string) { glog.Infoln(cmd) }

func bashCmd(cmd string) {
	glog.V(2).Infoln("executing bashCmd with:", cmd)
	cmds := strings.Split(cmd, " ")
	if len(cmds) < 1 {
		glog.Errorln("no compatible command passed to bashCmd")
		return
	}
	proc := exec.Command(cmds[0], cmds[1:]...)
	//TODO: Evaluate different cmdDir
	//TODO: Add a way to configure other writers (logFile?)
	proc.Stderr = os.Stderr
	proc.Stdout = os.Stdout
	err := proc.Run()
	if err != nil {
		glog.Errorln(err)
		return
	}
	glog.V(2).Infoln("finished bashCmd execution")
}

func newScheduler() (sched *scheduler.Scheduler, err error) {
	scPath := cfg.GetString("scheduler.path")
	schedule, err := scheduler.ReadSchedule(scPath)
	if err != nil {
		return
	}
	funcs := make(common.ScheduleFuncs)
	funcs["log"] = logCmd
	funcs["bash"] = bashCmd
	sched = scheduler.New(schedule, funcs)
	return
}

func newLogfile(logFolder string) *os.File {
	t := time.Now()
	logFileName := fmt.Sprintf("server_log_%v%d%v_%v-%v-%v.log", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	glog.Infoln("Creating Server Logfile: ", logFileName)
	_ = os.Mkdir(logFolder, 0775)
	logFile, err := os.OpenFile(path.Join(logFolder, logFileName), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return logFile
}

func getConfig() *viper.Viper {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.AddConfigPath(*configPathPtr)

	glog.V(1).Infof("Reading Config")

	err := cfg.ReadInConfig()
	if err != nil {
		message := fmt.Sprintf("Loading Config failed with Error: %v", err.Error())
		glog.Errorln(message)
		panic(message)
	}
	return cfg
}

//TODO: Move this to playnet common libs
func newLogger(dbg bool) *zap.Logger {
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
		zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
	)
	logger := zap.New(core)
	if dbg {
		logger = logger.WithOptions(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		)
	} else {
		logger = logger.WithOptions(
			zap.AddStacktrace(zap.FatalLevel),
		)
	}
	return logger
}
