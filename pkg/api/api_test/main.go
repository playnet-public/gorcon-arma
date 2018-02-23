package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/playnet-public/gorcon-arma/pkg/api"
	"github.com/playnet-public/libs/log"
)

func main() {
	l := log.NewNop()
	defer l.Close()
	a := api.New(l)
	go a.Run()
	time.Sleep(300 * time.Millisecond)
	resp, err := http.Get("http://127.0.0.1:80/api/test")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp.Status)
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(respData))
	resp, err = http.Get("http://127.0.0.1:80/api/example")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp.Status)
	respData, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(respData))
}
