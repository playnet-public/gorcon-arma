package main

import (
	"flag"
	"fmt"
	"runtime"

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

func main() {
	defer glog.Flush()
	glog.CopyStandardLogTo("info")
	flag.Parse()

	glog.Infof("Using %d go procs", *maxprocsPtr)
	runtime.GOMAXPROCS(*maxprocsPtr)

	if err := do(); err != nil {
		glog.Exit(err)
	}
}

func do() error {
	cfg := getConfig()

	// Placeholder for Log Test and Init Information

	// TODO: Refactor so scheduler and watcher are enabled seperately
	if cfg.GetBool("scheduler.enabled") {
		glog.Infof("Scheduler is enabled")
		schedulerPath := procwatch.SchedulePath(cfg.GetString("scheduler.path"))
		schedulerEntity, err := schedulerPath.Parse()
		if err != nil {
			return err
		}
		pwcfg := procwatch.Cfg{
			A3exe:    cfg.GetString("arma.path"),
			A3par:    cfg.GetString("arma.param"),
			Schedule: *schedulerEntity,
			//Timezone: cfg.GetInt("scheduler.timezone"),
		}

		watcher := procwatch.New(pwcfg)
		watcher.Start()
	} else {
		glog.Info("Scheduler is disabled")
	}

	for {
	}
	return nil
}

func getConfig() *viper.Viper {
	cfg := viper.New()
	cfg.SetConfigName("config")
	cfg.AddConfigPath(".")

	glog.V(2).Infof("Reading Config")

	err := cfg.ReadInConfig()
	if err != nil {
		message := fmt.Sprintf("Loading Config failed with Error: %v", err.Error())
		glog.Errorln(message)
		panic(message)
	}
	return cfg
}
