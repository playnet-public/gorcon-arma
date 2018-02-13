package funcs

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"

	"github.com/playnet-public/gorcon-arma/pkg/common"
	"github.com/playnet-public/gorcon-arma/pkg/rcon"
	"go.uber.org/zap"
)

//RegEx contains all common regular expressions for working with bercon strings
var RegEx = struct {
	PlayerID   *regexp.Regexp
	PlayerInfo *regexp.Regexp
	GUID       *regexp.Regexp
	NetInf     *regexp.Regexp
	Type       *regexp.Regexp
}{
	PlayerID:   regexp.MustCompile(`[Pp]layer\s#([0-9]+)\s`),
	PlayerInfo: regexp.MustCompile(`[Pp]layer\s#([0-9]+)\s(.+)`),
	GUID:       regexp.MustCompile(`([a-z|0-9]{32})`),
	NetInf:     regexp.MustCompile(`(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d+\b)`),
	Type:       regexp.MustCompile(`(disconnected|connected|Verified|GUID:|RCon)`),
	//connectReg, err := regexp.Compile(`[\S ]+\s#(\d)\s([\S ]+)\s\((\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})?:(\d+\b)\)\s[\S ]+`)
}

func (f RconFuncs) scanForPlayers(players *rcon.Players, r io.ReadCloser, quit chan error) {
	reg, err := regexp.Compile(`(\d+)\s+(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(\d+\b)\s+(\d+)\s+([0-9a-fA-F]+)\(\w+\)\s([\S ]+)`)
	if err != nil {
		quit <- err
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		errUnable := common.ErrUnableToParse
		f.log.Debug("applying regex", zap.String("type", "player"), zap.String("line", line))
		playerInfo := reg.FindStringSubmatch(line)
		if len(playerInfo) < 1 {
			continue
		}
		playerInfo = playerInfo[1:]
		f.log.Debug("player matched", zap.Strings("player", playerInfo))
		id, err := strconv.Atoi(playerInfo[0])
		if err != nil {
			quit <- err
		}
		ip := net.ParseIP(playerInfo[1])
		if ip == nil {
			quit <- errUnable
		}
		player := new(rcon.Player)
		player.ID = id
		player.Name = playerInfo[5]
		player.ExtID = playerInfo[4]
		player.IP = ip
		player.Port = playerInfo[2]
		player.Ping = playerInfo[3]

		players.Append(player)
	}
	if err := scanner.Err(); err != nil {
		quit <- err
	}
	quit <- nil
}

func (f RconFuncs) scanForBans(bans *rcon.Bans, r io.ReadCloser, quit chan error) {
	reg, err := regexp.Compile(`(\d+)\s+(\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}|[0-9a-fA-F]+)\s*([perm|\d]+)\s([\S ]+)`)
	if err != nil {
		quit <- err
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		f.log.Debug("applying regex", zap.String("type", "ban"), zap.String("line", line))
		banInfo := reg.FindStringSubmatch(line)
		if len(banInfo) < 1 {
			continue
		}
		if len(banInfo) < 4 {
			quit <- fmt.Errorf("Parsing Ban returned invalid length: %v", line)
		}
		banInfo = banInfo[1:]
		f.log.Debug("ban matched", zap.Strings("ban", banInfo))
		desc := banInfo[1]
		banType := "guid"
		if ip := net.ParseIP(banInfo[1]); ip != nil {
			f.log.Debug("ip ban detected", zap.Strings("ban", banInfo))
			banType = "ip"
		}
		ban := new(rcon.Ban)
		ban.Descriptor = desc
		ban.Type = banType
		//ban.Ends = banInfo[2]
		ban.Reason = banInfo[3]

		bans.Add(ban)
	}
	if err := scanner.Err(); err != nil {
		quit <- err
	}
	quit <- nil
}
