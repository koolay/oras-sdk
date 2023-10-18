package option

import (
	"fmt"
	"runtime"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Platform option struct.
type Platform struct {
	// request platform in the form of `os[/arch][/variant][:os_version]
	platform string
	Platform *ocispec.Platform
}

// parse parses the input platform flag to an oci platform type.
func (opts *Platform) Parse() error {
	if opts.platform == "" {
		return nil
	}

	// OS[/Arch[/Variant]][:OSVersion]
	// If Arch is not provided, will use GOARCH instead
	var platformStr string
	var p ocispec.Platform
	platformStr, p.OSVersion, _ = strings.Cut(opts.platform, ":")
	parts := strings.Split(platformStr, "/")
	switch len(parts) {
	case 3:
		p.Variant = parts[2]
		fallthrough
	case 2:
		p.Architecture = parts[1]
	case 1:
		p.Architecture = runtime.GOARCH
	default:
		return fmt.Errorf(
			"failed to parse platform %q: expected format os[/arch[/variant]]",
			opts.platform,
		)
	}
	p.OS = parts[0]
	if p.OS == "" {
		return fmt.Errorf("invalid platform: OS cannot be empty")
	}
	if p.Architecture == "" {
		return fmt.Errorf("invalid platform: Architecture cannot be empty")
	}
	opts.Platform = &p
	return nil
}
