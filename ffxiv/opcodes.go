package ffxiv

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnknownRegion = errors.New("ffxiv: unknown region")

//go:generate curl -o opcodes.json https://raw.githubusercontent.com/karashiiro/FFXIVOpcodes/master/opcodes.json
//go:embed opcodes.json
var opcodesJSON []byte

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

	ipcTypeCount = 6
)

func (t *OpcodeTable) GetOpcodeName(ipcType IpcType, opcode int) string {
	if mapping, ok := t.Lists[ipcType]; ok {
		if name, ok := mapping[opcode]; ok {
			return name
		}
	}

	return ""
}

func GetOpcodes(region Region) (OpcodeTable, error) {
	type rawOpcodeTable struct {
		Version string `json:"version"`
		Lists   map[IpcType][]struct {
			Name   string `json:"name"`
			Opcode int    `json:"opcode"`
		} `json:"lists"`
		Region Region `json:"region"`
	}

	// Deserialize the array of opcode tables from JSON
	var rawTables []rawOpcodeTable
	if err := json.Unmarshal(opcodesJSON, &rawTables); err != nil {
		return OpcodeTable{}, fmt.Errorf("unmarshal embedded opcodes: %w", err)
	}

	// Find the region we're looking for
	var desiredRawTable *rawOpcodeTable
	for i, t := range rawTables {
		if t.Region == region {
			desiredRawTable = &rawTables[i]
			break
		}
	}

	if desiredRawTable == nil {
		return OpcodeTable{}, fmt.Errorf("%w: %s", ErrUnknownRegion, region)
	}

	// Create a new OpcodeTable from the raw table
	table := OpcodeTable{
		Version: desiredRawTable.Version,
		Region:  desiredRawTable.Region,
		Lists:   make(map[IpcType]opcodeMapping, ipcTypeCount),
	}

	// Convert each IPC type list to a mapping (opcode -> name)
	for ipcType, list := range desiredRawTable.Lists {
		table.Lists[ipcType] = make(opcodeMapping, len(list))

		// Convert {name, opcode} structs to key-value pairs
		for _, def := range list {
			table.Lists[ipcType][def.Opcode] = def.Name
		}
	}

	return table, nil
}
