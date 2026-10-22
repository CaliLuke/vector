package workspacewebhooks

import (
	"net"
	"net/netip"
	"slices"

	"github.com/c-robinson/iplib/v2/iana"
)

var ipv6GlobalUnicast = netip.MustParsePrefix("2000::/3")

// The source policy blocks transition mechanisms even when IANA marks their
// outer addresses globally reachable. Membership comes from iplib's registry.
var blockedTransitionRFCs = []string{"RFC3056", "RFC4380", "RFC6052", "RFC8215"}

// iplib v2.0.5 predates these registry corrections. Keep the source runtime's
// observed decisions until a released registry update covers these entries.
// Upstream: https://github.com/c-robinson/iplib/issues/18
// Remove each correction when the pinned registry matches the source boundary corpus.
var ipv6RegistryCorrections = []struct {
	prefix  netip.Prefix
	blocked bool
}{
	{netip.MustParsePrefix("2001:5::/32"), true},
	{netip.MustParsePrefix("2001:30::/28"), false},
	{netip.MustParsePrefix("3fff::/20"), true},
}

func blockedIP(addr netip.Addr) bool {
	if !addr.IsValid() || !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.Is4In6() {
		return true
	}
	// Python's is_reserved rejects IPv6 outside the globally allocated unicast
	// space; netip.IsGlobalUnicast intentionally makes no allocation judgment.
	if addr.Is6() && !ipv6GlobalUnicast.Contains(addr) {
		return true
	}
	for _, correction := range ipv6RegistryCorrections {
		if correction.prefix.Contains(addr) {
			return correction.blocked
		}
	}
	longest := -1
	blocked := false
	for _, reservation := range iana.GetReservationsForIP(net.IP(addr.AsSlice())) {
		for _, rfc := range reservation.RFC {
			if slices.Contains(blockedTransitionRFCs, rfc) {
				return true
			}
		}
		bits, _ := reservation.Network.Mask().Size()
		// Specific public allocations override their reserved parent, including
		// 192.0.0.9 and 2001:1::1. Do not reject a public child with its parent.
		if bits > longest {
			longest, blocked = bits, !reservation.Global
		}
	}
	return blocked
}
