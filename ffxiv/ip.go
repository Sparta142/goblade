package ffxiv

import "net"

// DataCenterCIDRs is an array of all theorized public FINAL FANTASY XIV
// data center IP networks, in string CIDR notation.
//
// Found by resolving each lobby domain to its IPv4 address, then looking up
// the assigned address block that contains it in ARIN.
var DataCenterCIDRs = [...]string{
	// neolobby01.ffxiv.com, neolobby03.ffxiv.com, neolobby05.ffxiv.com
	"124.150.152.0/21",

	// neolobby02.ffxiv.com, neolobby04.ffxiv.com, neolobby08.ffxiv.com, neolobby11.ffxiv.com
	"204.0.0.0/14",

	// neolobby06.ffxiv.com, neolobby07.ffxiv.com
	"80.239.145.0/24",

	// neolobby09.ffxiv.com
	"153.254.80.0/22",

	// neolobby10.ffxiv.com
	"202.67.48.0/20",
}

// DataCenterNets is a list of all theorized public FINAL FANTASY XIV
// data center IP networks, as IPNets.
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

// Returns whether ip is probably a FINAL FANTASY XIV address.
func IsFinalFantasyIP(ip net.IP) bool {
	for _, ipnet := range DataCenterNets {
		if ipnet.Contains(ip) {
			return true
		}
	}

	return false
}
