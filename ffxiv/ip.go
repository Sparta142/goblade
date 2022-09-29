package ffxiv

import "net"

// DataCenterCIDRs is an array of all theorized public FINAL FANTASY XIV
// data center IP networks, in string CIDR notation.
//
// Based on data provided by https://is.xivup.com/adv.
var DataCenterCIDRs = [...]string{
	// Chaos (Europe)
	"80.239.145.79/32",
	"80.239.145.80/29",
	"80.239.145.88/31",

	// Light (Europe)
	"80.239.145.91/32",
	"80.239.145.92/30",
	"80.239.145.96/30",
	"80.239.145.100/31",

	// Elemental (Japan)
	"124.150.157.23/32",
	"124.150.157.24/29",
	"124.150.157.32/32",

	// Gaia (Japan)
	"124.150.157.38/31",
	"124.150.157.40/29",

	// Mana (Japan)
	"124.150.157.49/32",
	"124.150.157.50/31",
	"124.150.157.52/30",
	"124.150.157.56/31",
	"124.150.157.58/32",

	// Materia (Oceania)
	"153.254.80.75/32",
	"153.254.80.76/30",
	"153.254.80.80/31",
	"153.254.80.82/32",

	// Meteor (Japan)
	"202.67.52.205/32",
	"202.67.52.206/31",
	"202.67.52.208/30",
	"202.67.52.212/31",
	"202.67.52.214/32",

	// Aether (North America)
	"204.2.229.84/32",
	"204.2.229.86/32",
	"204.2.229.88/30",
	"204.2.229.92/32",

	// Primal (North America)
	"204.2.229.95/32",
	"204.2.229.96/30",
	"204.2.229.101/32",
	"204.2.229.102/31",

	// Crystal (North America)
	"204.2.229.106/31",
	"204.2.229.108/32",
	"204.2.229.110/31",
	"204.2.229.112/31",
	"204.2.229.114/32",
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
