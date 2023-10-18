package option

import (
	"fmt"
)

// DistributionSpec option struct.
type DistributionSpec struct {
	// ReferrersAPI indicates the preference of the implementation of the Referrers API.
	// Set to true for referrers API, false for referrers tag scheme, and nil for auto fallback.
	ReferrersAPI *bool

	// specFlag should be provided in form of`<version>-<api>-<option>`
	specFlag string
}

// Parse parses flags into the option.
func (opts *DistributionSpec) Parse() error {
	switch opts.specFlag {
	case "":
		opts.ReferrersAPI = nil
	case "v1.1-referrers-tag":
		isApi := false
		opts.ReferrersAPI = &isApi
	case "v1.1-referrers-api":
		isApi := true
		opts.ReferrersAPI = &isApi
	default:
		return fmt.Errorf("unknown distribution specification flag: %q", opts.specFlag)
	}
	return nil
}
