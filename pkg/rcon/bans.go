package rcon

import (
	"sync"
	"time"

	"fmt"

	"github.com/golang/glog"
)

//BanManager is responsible for handling Bans and their actions
type BanManager struct {
	Bans   Bans
	Checks []CheckFunc
}

//Ban represents an abstract rcon ban
type Ban struct {
	Descriptor string    `json:"desc"`
	Type       string    `json:"type"`
	Author     string    `json:"author"`
	Created    time.Time `json:"created"`
	Ends       time.Time `json:"ends"`
	Reason     string    `json:"reason"`
}

//Bans is the Ban List
type Bans struct {
	m map[string]*Ban
	sync.RWMutex
}

//CheckFunc defines the function interface used when passing in external check functions
type CheckFunc func(desc string) (bool, *Ban)

//Add Ban to Bans
func (b *Bans) Add(ban *Ban) {
	b.Lock()
	defer b.Unlock()
	if bl, ok := b.m[ban.Descriptor]; ok {
		glog.Warningf("Ban already exists for descriptor %v: %v - Overwriting with %v", ban.Descriptor, bl, ban)
	}
	b.m[ban.Descriptor] = ban
}

//Remove Ban from Bans
func (b *Bans) Remove(desc string) {
	b.Lock()
	defer b.Unlock()
	if _, ok := b.m[desc]; !ok {
		glog.Warningf("Ban does not exist for descriptor:", desc)
		return
	}
	delete(b.m, desc)
}

//Get Ban by Descriptor
func (b *Bans) Get(desc string) *Ban {
	b.RLock()
	defer b.RUnlock()
	bl, ok := b.m[desc]
	if ok {
		if bl.Descriptor != desc {
			glog.Errorf("Ban Map Descriptor Mismatch: Map(%v) - Ban(%v)", desc, bl.Descriptor)
			return nil
		}
		return bl
	}
	glog.Errorln("No Ban at Index", desc)
	return bl
}

//GetAll Bans
//NOTE: This function could be expensive and is not meant for every case
func (b *Bans) GetAll() (bans []*Ban) {
	b.RLock()
	defer b.RUnlock()
	for _, ba := range b.m {
		bans = append(bans, ba)
	}
	return
}

//NewBanManager returns a new Manager Object
func NewBanManager() *BanManager {
	bm := new(BanManager)
	bm.Bans.m = make(map[string]*Ban)
	return bm
}

//AddCheck to the given BanManager
func (bm *BanManager) AddCheck(check CheckFunc) {
	bm.Checks = append(bm.Checks, check)
}

//LoadBans from File at path
func (bm *BanManager) LoadBans(path string) {
	return
}

//SaveBans to file at path
func (bm *BanManager) SaveBans(path string) {
	return
}

//AddBan to Bans
func (bm *BanManager) AddBan(b *Ban) {
	bm.Bans.Lock()
	defer bm.Bans.Unlock()
	if _, ok := bm.Bans.m[b.Descriptor]; ok {
		glog.V(2).Infoln("overwriting ban with descriptor:", b.Descriptor)
	}
	bm.Bans.m[b.Descriptor] = b
}

//GetBan from Bans by desc
func (bm *BanManager) GetBan(desc string) *Ban {
	bm.Bans.RLock()
	defer bm.Bans.RUnlock()
	if b, ok := bm.Bans.m[desc]; ok {
		return b
	}
	return nil
}

//RemoveBan from Bans
func (bm *BanManager) RemoveBan(desc string) error {
	bm.Bans.Lock()
	defer bm.Bans.Unlock()
	if _, ok := bm.Bans.m[desc]; ok {
		return fmt.Errorf("failed to remove ban(%v) - no entry found", desc)
	}
	return nil
}

//CheckLocal checks the passed in desc against the local BanManager Bans
func (bm *BanManager) CheckLocal(desc string) (status bool, ban *Ban) {
	bm.Bans.RLock()
	defer bm.Bans.RUnlock()
	b, is := bm.Bans.m[desc]
	return is, b
}

//Check desc for a ban while starting every check as a single go routine to maintain non blocking behavior
func (bm *BanManager) Check(desc string) (status bool, ban *Ban) {
	if len(desc) < 5 {
		glog.Errorln("invalid CheckDesc", desc)
		return false, nil
	}
	if len(bm.Checks) < 1 {
		glog.Warningln("no ban checks defined")
		return false, nil
	}
	glog.V(2).Infoln("checking Player Descriptor", desc)
	bans := make(chan *Ban, len(bm.Checks))
	wg := sync.WaitGroup{}
	wg.Add(len(bm.Checks))
	glog.V(3).Infof("starting %d checkFuncs", len(bm.Checks))
	for _, f := range bm.Checks {
		go func(f CheckFunc) {
			is, b := f(desc)
			if is {
				bans <- b
			}
			wg.Done()
		}(f)
	}

	go func() {
		wg.Wait()
		close(bans)
	}()

	for ban = range bans {
		//glog.V(2).Infoln("handling ban check result", ban)
		if ban != nil {
			return true, ban
		}
	}
	glog.V(2).Infof("ban check without result, player(%v) OK", desc)
	return false, nil
}
