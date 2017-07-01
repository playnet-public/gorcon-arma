package banManager

import (
	"fmt"
	"io"
	"testing"

	"github.com/playnet-public/gorcon-arma/rcon"
)

func Test_Refresh(t *testing.T) {
	pm := NewBanManager()
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
	fmt.Println(pm.Bans)
}

func dummyExec(cmd []byte, w io.WriteCloser) error {
	cmds := string(cmd)
	switch cmds {
	case "bans":
		w.Write([]byte("GUID Bans:\n"))
		w.Write([]byte("[#] [GUID] [Minutes left] [Reason]\n"))
		w.Write([]byte("----------------------------------------\n"))
		w.Write([]byte("0  00c4406796d6d40ea5da9b701327f202 perm Mystic\n"))
		w.Write([]byte("1  e82445f009b9d63371ce19190f5d2661 perm Mystic\n"))
		w.Write([]byte("2  f1b640ec28d350b484c55ca30bb4b6b2 perm Monky\n"))
		w.Write([]byte("3  6347e8acd55ad2506f88f430de9e5e57 perm Scriptkid\n"))
		w.Write([]byte("4  01b813f2de3351560b41b66ac94188cc perm Scriptkid\n"))
		w.Write([]byte("5  eca25cd3ecafb63ba5a21b6e0b13b69d perm Scriptkid\n"))
		w.Write([]byte("6  22d83cd06a383f3eda3c34631a6ebd03 perm Scriptkid\n"))
		w.Write([]byte("7  388d8a4b04e281ea6f7f33cd6c058a9b perm Scriptkid\n"))
		w.Write([]byte("8  497f071e76afcc1e662101b807c8954e perm Scriptkid\n"))
		w.Write([]byte("9  97ee898ff149e746694624e9c4140aa5 perm Scriptkid\n"))
		w.Write([]byte("10 c8e02a00328a46d300f278be9d3b423d perm Scriptkid\n"))
		w.Write([]byte("11 c3a61d4fca69c45498850410a855cdf3 perm Came\n"))
		w.Write([]byte("12 f740ad1ad28e10c3cf02e245eed2efb8 perm Came\n"))
		w.Write([]byte("13 f03870375f7b1bf172e6306d5c808e2b perm Scriptkid\n"))
		w.Write([]byte("14 349e397e0a3650e82b26a09b196979e0 perm Admin Ban\n"))
		w.Write([]byte("\n"))
		w.Write([]byte("IP Bans:\n"))
		w.Write([]byte("[#] [IP Address] [Minutes left] [Reason]\n"))
		w.Write([]byte("----------------------------------------------\n"))
		w.Write([]byte("115 109.156.210.166 perm Mystic\n"))
		w.Write([]byte("116 173.80.124.105  perm Scriptkid\n"))
		w.Write([]byte("117 173.80.119.222  perm Scriptkid\n"))
		w.Write([]byte("118 66.87.94.0      perm Scriptkid\n"))
		w.Write([]byte("119 66.87.92.80     perm Scriptkid\n"))
		w.Write([]byte("120 173.80.107.33   perm Scriptkid\n"))
		w.Write([]byte("121 101.160.152.154 perm Scriptkid\n"))
		w.Write([]byte("122 101.160.51.166  perm Scriptkid\n"))
		w.Close()
	}
	return nil
}
