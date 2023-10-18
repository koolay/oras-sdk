package artifacts

import (
	"context"
	"errors"
	"fmt"
	"sync"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/file"

	"github.com/koolay/oras-sdk/display"
	"github.com/koolay/oras-sdk/option"
)

type PullOptions struct {
	option.Cache
	option.Common
	option.Platform
	option.Target

	concurrency       int
	KeepOldFiles      bool
	IncludeSubject    bool
	PathTraversal     bool
	Output            string
	ManifestConfigRef string
}

func RunPull(ctx context.Context, opts PullOptions) error {
	ctx, logger := opts.WithContext(ctx)
	// Copy Options
	var printed sync.Map
	copyOptions := oras.DefaultCopyOptions
	copyOptions.Concurrency = opts.concurrency
	var err error
	if opts.Platform.Platform != nil {
		copyOptions.WithTargetPlatform(opts.Platform.Platform)
	}

	target, err := opts.NewReadonlyTarget(ctx, opts.Common, logger)
	if err != nil {
		return err
	}
	if err := opts.EnsureReferenceNotEmpty(); err != nil {
		return err
	}
	src, err := opts.CachedTarget(target)
	if err != nil {
		return err
	}
	dst, err := file.New(opts.Output)
	if err != nil {
		return err
	}
	defer dst.Close()
	dst.AllowPathTraversalOnWrite = opts.PathTraversal
	dst.DisableOverwrite = opts.KeepOldFiles

	pulledEmpty := true
	copyOptions.PreCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		if _, ok := printed.LoadOrStore(generateContentKey(desc), true); ok {
			return nil
		}
		return display.PrintStatus(desc, "Downloading", opts.Verbose)
	}
	copyOptions.PostCopy = func(ctx context.Context, desc ocispec.Descriptor) error {
		// restore named but deduplicated successor nodes
		successors, err := content.Successors(ctx, dst, desc)
		if err != nil {
			return err
		}
		for _, s := range successors {
			if _, ok := s.Annotations[ocispec.AnnotationTitle]; ok {
				if err := printOnce(&printed, s, "Restored   ", opts.Verbose); err != nil {
					return err
				}
			}
		}
		name, ok := desc.Annotations[ocispec.AnnotationTitle]
		if !ok {
			if !opts.Verbose {
				return nil
			}
			name = desc.MediaType
		} else {
			// named content downloaded
			pulledEmpty = false
		}
		printed.Store(generateContentKey(desc), true)
		return display.Print("Downloaded ", display.ShortDigest(desc), name)
	}

	desc, err := oras.Copy(ctx, src, opts.Reference, dst, opts.Reference, copyOptions)
	if err != nil {
		if errors.Is(err, file.ErrPathTraversalDisallowed) {
			err = fmt.Errorf(
				"%s: %w",
				"use flag --allow-path-traversal to allow insecurely pulling files outside of working directory",
				err,
			)
		}
		return err
	}
	if pulledEmpty {
		fmt.Println("Downloaded empty artifact")
	}
	fmt.Println("Pulled", opts.AnnotatedReference())
	fmt.Println("Digest:", desc.Digest)
	return nil
}

// generateContentKey generates a unique key for each content descriptor, using
// its digest and name if applicable.
func generateContentKey(desc ocispec.Descriptor) string {
	return desc.Digest.String() + desc.Annotations[ocispec.AnnotationTitle]
}

func printOnce(printed *sync.Map, s ocispec.Descriptor, msg string, verbose bool) error {
	if _, loaded := printed.LoadOrStore(generateContentKey(s), true); loaded {
		return nil
	}
	return display.PrintStatus(s, msg, verbose)
}
