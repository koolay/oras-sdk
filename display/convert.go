package display

import (
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// ShortDigest converts the digest of the descriptor to a short form for displaying.
func ShortDigest(desc ocispec.Descriptor) (digestString string) {
	digestString = desc.Digest.String()
	if err := desc.Digest.Validate(); err == nil {
		if algo := desc.Digest.Algorithm(); algo == digest.SHA256 {
			digestString = desc.Digest.Encoded()[:12]
		}
	}
	return digestString
}
