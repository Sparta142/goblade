package ffxiv

import "net"

// DataCenterCIDRs is an array of all known public FINAL FANTASY XIV
// data center IP networks, in string CIDR notation.
var DataCenterCIDRs = [...]string{
	"204.2.229.0/24",   // North America: Aether, Crystal, Primal
	"195.82.50.0/24",   // Europe: Chaos, Light
	"124.150.157.0/24", // Japan: Elemental, Gaia, Mana
	"153.254.80.0/24",  // Oceania: Materia
}

// DataCenterNets is a list of all known public
// FINAL FANTASY XIV data center IP networks, as IPNets.
var DataCenterNets = make([]net.IPNet, len(DataCenterCIDRs))

// Returns whether ip is a known FINAL FANTASY XIV address.
func IsFinalFantasyIP(ip net.IP) bool {
	for _, ipnet := range DataCenterNets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}

func init() {
	for i, s := range DataCenterCIDRs {
		_, net, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		DataCenterNets[i] = *net
	}
}
