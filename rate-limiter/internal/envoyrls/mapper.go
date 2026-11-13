// mapper.go — extracts and validates the client IP from Envoy's descriptor.

package envoyrls

import (
	"fmt"
	"net/netip"
	"strings"

	ratelimitcommonv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
)

const remoteAddressDescriptorKey = "remote_address"

func ExtractClientIP(descriptor *ratelimitcommonv3.RateLimitDescriptor) (string, error) {
	if descriptor == nil {
		return "", fmt.Errorf("rate-limit descriptor must not be nil")
	}

	var clientIP string

	for _, entry := range descriptor.GetEntries() {
		if entry == nil || entry.GetKey() != remoteAddressDescriptorKey {
			continue
		}

		// Reject duplicate remote_address entries.
		if clientIP != "" {
			return "", fmt.Errorf("duplicate remote_address descriptor")
		}

		clientIP = strings.TrimSpace(entry.GetValue())
	}

	if clientIP == "" {
		return "", fmt.Errorf("remote_address descriptor is missing")
	}

	addr, err := netip.ParseAddr(clientIP)
	if err != nil {
		return "", fmt.Errorf("invalid client IP address %q: %w", clientIP, err)
	}

	// Normalize IPv4-mapped IPv6 addresses.
	return addr.Unmap().String(), nil
}
