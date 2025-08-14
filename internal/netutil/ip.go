package netutil

import (
	"log/slog"
	"net"
)

// IsAllowed checks if a given IP address is present in a list of allowed subnets/IPs.
// The list can contain single IP addresses (e.g., "192.168.1.50") or CIDR blocks (e.g., "10.0.0.0/8").
// If the allowed list is empty, all IPs are considered allowed.
func IsAllowed(log *slog.Logger, clientIP net.IP, allowedSubnets []string) bool {
	// If the list is empty, access is not restricted.
	if len(allowedSubnets) == 0 {
		return true
	}

	for _, subnetStr := range allowedSubnets {
		// Try parsing as a CIDR block first.
		_, cidrNet, err := net.ParseCIDR(subnetStr)
		if err == nil {
			// If it's a valid CIDR, check if the IP is contained within it.
			if cidrNet.Contains(clientIP) {
				log.Debug("client IP is within allowed CIDR", "clientIP", clientIP.String(), "cidr", subnetStr)
				return true
			}
			continue
		}

		// If CIDR parsing failed, try parsing as a single IP address.
		ip := net.ParseIP(subnetStr)
		if ip != nil {
			// If it's a valid IP, check for an exact match.
			if ip.Equal(clientIP) {
				log.Debug("client IP is an exact match for allowed IP", "clientIP", clientIP.String(), "allowedIP", subnetStr)
				return true
			}
			continue
		}

		// If the entry is neither valid CIDR nor a valid IP, log a warning.
		log.Warn("invalid entry in AllowedSubnets list", "entry", subnetStr)
	}

	// If no match was found after checking all rules, deny access.
	log.Warn("client IP is not in any allowed subnet", "clientIP", clientIP.String())
	return false
}
