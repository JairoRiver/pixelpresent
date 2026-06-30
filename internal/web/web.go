// Package web embeds the built Astro frontend so the Go binary serves the static
// site directly, with no external files in production. The contents of dist are
// produced by `task build` (Astro builds into this directory); only the .gitkeep
// placeholder is committed, so the package still compiles on a fresh checkout
// even before the frontend has been built.
package web

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// ErrFrontendNotBuilt reports that the embedded dist holds only the placeholder
// (no real site): the binary was compiled without building the frontend first.
var ErrFrontendNotBuilt = errors.New("embedded frontend not built")

// Dist returns the embedded static site rooted at the build output directory. It
// returns ErrFrontendNotBuilt if the build output is missing (index.html is not
// present), i.e. the binary was compiled without running `task build` — so the
// caller can log a clear, actionable message instead of serving silent 404s.
func Dist() (fs.FS, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil, fmt.Errorf("%w (index.html not found in embed): build it with `task build`", ErrFrontendNotBuilt)
	}
	return sub, nil
}
