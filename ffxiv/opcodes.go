package ffxiv

import (
	_ "embed"
	"encoding/json"

	log "github.com/sirupsen/logrus"
)

//go:generate curl -o opcodes.json https://raw.githubusercontent.com/karashiiro/FFXIVOpcodes/master/opcodes.json
//go:embed opcodes.json
var opcodesJSON []byte

var opcodeTables map[Region]OpcodeTable

type Region string
type IpcType string
type opcodeMapping map[int]string

type OpcodeTable struct {
	Version string
	Lists   map[IpcType]opcodeMapping
	Region  Region
}

const (
	RegionGlobal = Region("Global")
	RegionChina  = Region("CN")
	RegionKorea  = Region("KR")
)

const (
	ServerZoneIpcType  = IpcType("ServerZoneIpcType")
	ClientZoneIpcType  = IpcType("ClientZoneIpcType")
	ServerLobbyIpcType = IpcType("ServerLobbyIpcType")
	ClientLobbyIpcType = IpcType("ClientLobbyIpcType")
	ServerChatIpcType  = IpcType("ServerChatIpcType")
	ClientChatIpcType  = IpcType("ClientChatIpcType")
)

func GetOpcodeTable(region Region) (v OpcodeTable, ok bool) {
	v, ok = opcodeTables[region]
	return
}

func (t *OpcodeTable) GetOpcodeName(ipcType IpcType, opcode int) string {
	if mapping, ok := t.Lists[ipcType]; ok {
		if name, ok := mapping[opcode]; ok {
			return name
		}
	}
	return ""
}

func init() {
	var err error
	if opcodeTables, err = parseOpcodes(); err != nil {
		log.WithError(err).Fatal("Failed to unmarshal embedded opcodes")
	}

	log.Debugf("Loaded embedded opcode definitions for %d regions", len(opcodeTables))
}

func parseOpcodes() (map[Region]OpcodeTable, error) {
	type rawOpcodeTable struct {
		Version string `json:"version"`
		Lists   map[IpcType][]struct {
			Name   string `json:"name"`
			Opcode int    `json:"opcode"`
		} `json:"lists"`
		Region Region `json:"region"`
	}

	// Load the JSON data into memory
	var rawTables []rawOpcodeTable
	if err := json.Unmarshal(opcodesJSON, &rawTables); err != nil {
		return nil, err
	}

	parsedTables := make(map[Region]OpcodeTable, len(rawTables))

	// Loop through all regions
	for _, rawTable := range rawTables {
		parsedTable := OpcodeTable{
			Version: rawTable.Version,
			Lists:   make(map[IpcType]opcodeMapping, 6),
			Region:  rawTable.Region,
		}

		// Convert each IPC type list to a mapping (opcode -> name)
		for typ, list := range rawTable.Lists {
			parsedTable.Lists[typ] = make(opcodeMapping, len(list))

			// Convert {name, opcode} objects to key-value pairs
			for _, def := range list {
				parsedTable.Lists[typ][def.Opcode] = def.Name
			}
		}

		parsedTables[rawTable.Region] = parsedTable
	}

	return parsedTables, nil
}
