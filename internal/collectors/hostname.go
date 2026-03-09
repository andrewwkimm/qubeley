// resolveHostname returns the system hostname, falling back to "unknown" if
// it cannot be determined.
package collectors

import "os"

func resolveHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
