package ffxiv

import (
	_ "embed"
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

//go:generate curl -o opcodes.json https://raw.githubusercontent.com/karashiiro/FFXIVOpcodes/master/opcodes.json
//go:embed opcodes.json
var opcodesJson []byte

var opcodes []struct {
	Version string `json:"version"`
	Region  string `json:"region"`
	Lists   struct {
		ServerZone  []opcodeDef `json:"ServerZoneIpcType"`
		ClientZone  []opcodeDef `json:"ClientZoneIpcType"`
		ServerLobby []opcodeDef `json:"ServerLobbyIpcType"`
		ClientLobby []opcodeDef `json:"ClientLobbyIpcType"`
		ServerChat  []opcodeDef `json:"ServerChatIpcType"`
		ClientChat  []opcodeDef `json:"ClientChatIpcType"`
	} `json:"lists"`
}

type opcodeDef struct {
	Name   string `json:"name"`
	Opcode int    `json:"opcode"`
}

func init() {
	if err := json.Unmarshal(opcodesJson, &opcodes); err != nil {
		log.WithError(err).Fatal("Failed to unmarshal embedded opcodes")
	}

	log.Infof("Loaded embedded opcode definitions for %d regions", len(opcodes))
}
