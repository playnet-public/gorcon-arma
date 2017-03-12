package main

import (
	"flag"
	"fmt"
	"runtime"

	"play-net.org/gorcon-arma/rcon"

	"net"

	"bufio"
	"io"

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
	glog.Infof("Using Server IP: %s", cfg.GetString("arma.ip"))
	glog.Infof("Using Server Port: %s", cfg.GetString("arma.port"))
	udpadr, err := net.ResolveUDPAddr("udp", cfg.GetString("arma.ip")+":"+cfg.GetString("arma.port"))
	if err != nil {
		glog.Errorln("Could not convert ArmA IP and Port")
		return err
	}
	becfg := rcon.Config{
		Addr:           udpadr,
		Password:       cfg.GetString("arma.password"),
		KeepAliveTimer: 10,
	}

	client := rcon.New(becfg)

	r, w := io.Pipe()
	client.SetEventWriter(w)
	client.SetChatWriter(w)

	err = client.Connect()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fmt.Printf("RCON: %s", scanner.Text())
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
