package playerManager

import (
	"fmt"
	"io"
	"testing"

	"github.com/playnet-public/gorcon-arma/rcon"
)

func Test_Refresh(t *testing.T) {
	pm := NewPlayerManager()
	pm.Client = rcon.NewClient(
		nil,
		nil,
		dummyExec,
		nil,
		nil,
	)
	err := pm.Refresh()
	if err != nil {
		t.Error(err)
	}
	fmt.Println(pm.Players)
}

func dummyExec(cmd []byte, w io.WriteCloser) error {
	cmds := string(cmd)
	switch cmds {
	case "players":
		w.Write([]byte("Players on server:\n"))
		w.Write([]byte("[#] [IP Address]:[Port] [Ping] [GUID] [Name]\n"))
		w.Write([]byte("--------------------------------------------------\n"))
		w.Write([]byte("0   95.157.20.219:2304    47   7061d3595e0bf5e483e0d48584d649ad(OK) ExoTic\n"))
		w.Write([]byte("(1 players in total)\n"))
		w.Close()
	}
	return nil
}
