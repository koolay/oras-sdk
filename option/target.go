package option

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"golang.org/x/exp/slog"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
)

const (
	TargetTypeRemote    = "registry"
	TargetTypeOCILayout = "oci-layout"
)

// Target struct contains flags and arguments specifying one registry or image
// layout.
type Target struct {
	Remote
	RawReference string
	Type         string
	Reference    string // contains tag or digest
	// Path contains
	//  - path to the OCI image layout target, or
	//  - registry and repository for the remote target
	Path string

	isOCILayout bool
}

// AnnotatedReference returns full printable reference.
func (opts *Target) AnnotatedReference() string {
	return fmt.Sprintf("[%s] %s", opts.Type, opts.RawReference)
}

// Parse gets target options from user input.
func (opts *Target) Parse() error {
	switch {
	case opts.isOCILayout:
		opts.Type = TargetTypeOCILayout
		if len(opts.headerFlags) != 0 {
			return errors.New("custom header flags cannot be used on an OCI image layout target")
		}
		return nil
	default:
		opts.Type = TargetTypeRemote
		return opts.Remote.Parse()
	}
}

// parseOCILayoutReference parses the raw in format of <path>[:<tag>|@<digest>]
func parseOCILayoutReference(raw string) (path string, ref string, err error) {
	if idx := strings.LastIndex(raw, "@"); idx != -1 {
		// `digest` found
		path = raw[:idx]
		ref = raw[idx+1:]
	} else {
		// find `tag`
		path, ref, err = parseFileRef(raw, "")
	}
	return
}

// NewTarget generates a new target based on opts.
func (opts *Target) NewTarget(common Common, logger *slog.Logger) (oras.GraphTarget, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		var err error
		opts.Path, opts.Reference, err = parseOCILayoutReference(opts.RawReference)
		if err != nil {
			return nil, err
		}
		return oci.New(opts.Path)
	case TargetTypeRemote:
		repo, err := opts.NewRepository(opts.RawReference, common, logger)
		if err != nil {
			return nil, err
		}
		tmp := repo.Reference
		tmp.Reference = ""
		opts.Path = tmp.String()
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// ReadOnlyGraphTagFinderTarget represents a read-only graph target with tag
// finder capability.
type ReadOnlyGraphTagFinderTarget interface {
	oras.ReadOnlyGraphTarget
	registry.TagLister
}

// NewReadonlyTargets generates a new read only target based on opts.
func (opts *Target) NewReadonlyTarget(
	ctx context.Context,
	common Common,
	logger *slog.Logger,
) (ReadOnlyGraphTagFinderTarget, error) {
	switch opts.Type {
	case TargetTypeOCILayout:
		var err error
		opts.Path, opts.Reference, err = parseOCILayoutReference(opts.RawReference)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(opts.Path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return oci.NewFromFS(ctx, os.DirFS(opts.Path))
		}
		return oci.NewFromTar(ctx, opts.Path)
	case TargetTypeRemote:
		repo, err := opts.NewRepository(opts.RawReference, common, logger)
		if err != nil {
			return nil, err
		}
		tmp := repo.Reference
		tmp.Reference = ""
		opts.Path = tmp.String()
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// EnsureReferenceNotEmpty ensures whether the tag or digest is empty.
func (opts *Target) EnsureReferenceNotEmpty() error {
	if opts.Reference == "" {
		return errors.New("reference cannot be empty")
	}
	return nil
}

// BinaryTarget struct contains flags and arguments specifying two registries or
// image layouts.
type BinaryTarget struct {
	From        Target
	To          Target
	resolveFlag []string
}

// EnableDistributionSpecFlag set distribution specification flag as applicable.
func (opts *BinaryTarget) EnableDistributionSpecFlag() {
	opts.From.EnableDistributionSpecFlag()
	opts.To.EnableDistributionSpecFlag()
}

// Parse parses user-provided flags and arguments into option struct.
func (opts *BinaryTarget) Parse() error {
	opts.From.warned = make(map[string]*sync.Map)
	opts.To.warned = opts.From.warned
	// resolve are parsed in array order, latter will overwrite former
	opts.From.resolveFlag = append(opts.resolveFlag, opts.From.resolveFlag...)
	opts.To.resolveFlag = append(opts.resolveFlag, opts.To.resolveFlag...)
	return Parse(opts)
}

// parseFileRef parses file reference on unix.
func parseFileRef(reference string, defaultMetadata string) (filePath, metadata string, err error) {
	i := strings.LastIndex(reference, ":")
	if i < 0 {
		filePath, metadata = reference, defaultMetadata
	} else {
		filePath, metadata = reference[:i], reference[i+1:]
	}
	if filePath == "" {
		return "", "", fmt.Errorf("found empty file path in %q", reference)
	}
	return filePath, metadata, nil
}
