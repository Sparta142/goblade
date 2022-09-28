package ffxiv

import "net"

// DataCenterCIDRs is an array of all theorized public FINAL FANTASY XIV
// data center IP networks, in string CIDR notation.
//
// Based on data provided by https://is.xivup.com/adv.
var DataCenterCIDRs = [...]string{
	"204.2.229.64/26",  // North America: Aether, Crystal, Primal
	"195.82.50.32/27",  // Europe: Chaos, Light
	"124.150.157.0/26", // Japan: Elemental, Gaia, Mana
	"202.67.52.192/26", // Japan: Meteor
	"153.254.80.64/27", // Oceania: Materia
}

// DataCenterNets is a list of all theorized public
// FINAL FANTASY XIV data center IP networks, as IPNets.
var DataCenterNets = func() []net.IPNet {
	nets := make([]net.IPNet, len(DataCenterCIDRs))

	for i, s := range DataCenterCIDRs {
		_, net, err := net.ParseCIDR(s)
		if err != nil {
			panic(err)
		}
		nets[i] = *net
	}

	return nets
}()

// Returns whether ip is a known FINAL FANTASY XIV address.
func IsFinalFantasyIP(ip net.IP) bool {
	for _, ipnet := range DataCenterNets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}
