package ffxiv

import "net"

// All known public FINAL FANTASY XIV data center IP networks, in CIDR notation.
var DataCenterCIDRs = [...]string{
	"204.2.229.0/24",   // North America: Aether, Crystal, Primal
	"195.82.50.0/24",   // Europe: Chaos, Light
	"124.150.157.0/24", // Japan: Elemental, Gaia, Mana
	"153.254.80.0/24",  // Oceania: Materia
}

// All known public FINAL FANTASY XIV data center IP networks.
var DataCenterNets []net.IPNet

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
	DataCenterNets = make([]net.IPNet, len(DataCenterCIDRs))

	for i, s := range DataCenterCIDRs {
		_, net, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		DataCenterNets[i] = *net
	}
}
