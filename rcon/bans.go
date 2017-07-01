package rcon

//BanManager is responsible for handling Bans and their actions
type BanManager struct {
	Bans    Bans
	Refresh refreshBans
	Get     getBans
	Save    saveBans
	Load    loadBans
	Add     addBan
	Remove  removeBan
}

//Ban represents an abstract rcon ban
type Ban struct {
	ID         int    `json:"id"`
	Descriptor string `json:"desc"`
	Type       string `json:"type"`
	Duration   string `json:"duration"`
	Reason     string `json:"reason"`
}

//Bans is the Ban List
type Bans []Ban

type refreshBans func() error
type getBans func() Bans
type saveBans func() error
type loadBans func() error
type addBan func(p Ban) error
type removeBan func(p Ban) error

//NewBanManager returns a new Manager Object
func NewBanManager(
	refresh refreshBans,
	get getBans,
	save saveBans,
	load loadBans,
	add addBan,
	remove removeBan,
) *BanManager {
	pm := new(BanManager)
	pm.Refresh = refresh
	pm.Get = get
	pm.Save = save
	pm.Load = load
	pm.Add = add
	pm.Remove = remove
	return pm
}
