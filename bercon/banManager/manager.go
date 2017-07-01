package banManager

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"

	"regexp"

	raven "github.com/getsentry/raven-go"
	"github.com/golang/glog"
	"github.com/playnet-public/gorcon-arma/rcon"
)

//BanManager is responsible for handling Bans and their actions
type BanManager struct {
	Bans   rcon.Bans
	Client *rcon.Client
}

//NewBanManager returns a new Manager Object
func NewBanManager() *BanManager {
	pm := new(BanManager)
	return pm
}

//Refresh the Bans List
//TODO: This looks way to dirty, maybe look for better solutions
func (pm *BanManager) Refresh() error {

	r, w := io.Pipe()

	bans := new(rcon.Bans)
	quit := make(chan error)

	go scanForBans(bans, r, quit)

	//Fetch Ban List from RCon
	err := pm.Client.Exec([]byte("bans"), w)
	if err != nil {
		return err
	}
	q := <-quit
	if q == nil {
		pm.Bans = *bans
		return nil
	}
	return q
}

//Get returns all bans
func (pm *BanManager) Get() rcon.Bans {
	return pm.Bans
}

func scanForBans(bans *rcon.Bans, r io.ReadCloser, quit chan error) {
	reg, err := regexp.Compile(`(\d+)\s+(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|[0-9a-fA-F]+)\s*([perm|\d]+)\s([\S ]+)`)
	if err != nil {
		quit <- err
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		glog.V(3).Infoln("Applying Regex onto:", line)
		banInfo := reg.FindStringSubmatch(line)
		if len(banInfo) < 1 {
			continue
		}
		if len(banInfo) < 4 {
			quit <- fmt.Errorf("Parsing Ban returned invalid length: %v", line)
		}
		banInfo = banInfo[1:]
		glog.V(3).Infoln("Ban Matched:", banInfo)
		id, err := strconv.Atoi(banInfo[0])
		if err != nil {
			quit <- err
		}
		desc := banInfo[1]
		banType := "guid"
		if ip := net.ParseIP(banInfo[1]); ip != nil {
			glog.V(2).Infoln("Ban detected as IP Ban")
			banType = "ip"
		}
		ban := rcon.Ban{
			ID:         id,
			Descriptor: desc,
			Type:       banType,
			Duration:   banInfo[2],
			Reason:     banInfo[3],
		}
		*bans = append(*bans, ban)
	}
	if err := scanner.Err(); err != nil {
		quit <- err
	}
	quit <- nil
}

//Add the passed in Ban for duration (in minutes)
func (pm *BanManager) Add(p rcon.Ban) error {
	if p.Type == "guid" {
		cmd := fmt.Sprintf("addBan %v %v %v", p.Descriptor, p.Duration, p.Reason)
		glog.V(2).Infoln("Sending Command:", cmd)
		return pm.Client.Exec([]byte(cmd), os.Stdout)
	}
	err := errors.New("Adding IP Bans is not implemented yet")
	raven.CaptureError(err, map[string]string{"app": "rcon", "module": "client"})
	glog.Warningln(err)
	return err
}

//Remove the passed in Ban
func (pm *BanManager) Remove(p rcon.Ban) error {
	cmd := fmt.Sprintln("removeBan", p.ID)
	glog.V(2).Infoln("Sending Command:", cmd)
	return pm.Client.Exec([]byte(cmd), os.Stdout)
}

//Save all bans to file
func (pm *BanManager) Save() error {
	return pm.Client.Exec([]byte("writeBans"), os.Stdout)
}

//Load bans
func (pm *BanManager) Load() error {
	return pm.Client.Exec([]byte("loadBans"), os.Stdout)
}
