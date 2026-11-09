
package redisstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/netip"
	"strings"
)

const defaultKeyPrefix = "ratelimit"

// KeyBuilder generates Redis keys for rate-limiting state.
type KeyBuilder struct {
	prefix string
}

// NewKeyBuilder creates a Redis key builder.
//
// If prefix is empty, the default "ratelimit" prefix is used.
func NewKeyBuilder(prefix string) *KeyBuilder {
	prefix = strings.TrimSpace(prefix)

	if prefix == "" {
		prefix = defaultKeyPrefix
	}

	return &KeyBuilder{
		prefix: prefix,
	}
}

// WindowKey generates the Redis key used to store an IP's request count.
func (b *KeyBuilder) WindowKey(ip string) (string, error) {
	hash, err := hashIP(ip)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:{%s}:window", b.prefix, hash), nil
}

// BlockKey generates the Redis key used to store an IP's block status.
func (b *KeyBuilder) BlockKey(ip string) (string, error) {
	hash, err := hashIP(ip)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:{%s}:block", b.prefix, hash), nil
}

// hashIP validates and normalizes an IP address before hashing it.
func hashIP(ip string) (string, error) {
	addr, err := netip.ParseAddr(strings.TrimSpace(ip))
	if err != nil {
		return "", fmt.Errorf("invalid IP address %q: %w", ip, err)
	}

	// Normalize IPv4-mapped IPv6 addresses.
	addr = addr.Unmap()

	sum := sha256.Sum256([]byte(addr.String()))

	return hex.EncodeToString(sum[:]), nil
}
