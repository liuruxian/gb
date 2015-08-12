package gb

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Install stores a copy of the compiled file in Project.Pkgdir
func Install(pkg *Package, t PkgTarget) PkgTarget {
	if pkg.SkipInstall {
		return t
	}
	if pkg.Scope == "test" {
		Debugf("%v is test scoped, not installing", pkg)
		return t
	}
	i := install{
		PkgTarget: t,
		dest:      pkgfile(pkg),
	}
	i.target = newTarget(i.install, t)
	return &i
}

// cachePackage returns a PkgTarget representing the cached output of pkg.
type cachedPackage struct {
	pkg *Package
}

func (c *cachedPackage) Pkgfile() string {
	return pkgfile(c.pkg)
}

func (c *cachedPackage) String() string {
	return fmt.Sprintf("cached %v", c.pkg.ImportPath)
}

func (c *cachedPackage) Result() error {
	// TODO(dfc) _, err := os.Stat(c.Pkgfile())
	return nil
}

type cachedTarget struct {
	target Target
}

func (c *cachedTarget) String() string {
	return fmt.Sprintf("cached %v", c.target)
}

func (c *cachedTarget) Result() error {
	return nil
}

type install struct {
	target
	PkgTarget
	dest string
}

func (i *install) String() string {
	return fmt.Sprintf("cache %v", i.PkgTarget)
}

func (i *install) install() error {
	return copyfile(i.dest, i.Pkgfile())
}

func (i *install) Result() error {
	return i.target.Result()
}

// pkgdir returns the destination for object cached for this Package.
func pkgdir(pkg *Package) string {
	if pkg.Scope == "test" {
		panic("pkgdir called with test scope")
	}
	return filepath.Join(pkg.Pkgdir(), filepath.Dir(filepath.FromSlash(pkg.ImportPath)))
}

func pkgfile(pkg *Package) string {
	return filepath.Join(pkgdir(pkg), filepath.Base(filepath.FromSlash(pkg.ImportPath))+".a")
}

// isStale returns true if the source pkg is considered to be stale with
// respect to its installed version.
func isStale(pkg *Package) bool {
	if pkg.Force {
		return true
	}

	// tests are always stale, they are never installed
	if pkg.Scope == "test" {
		return true
	}

	// Package is stale if completely unbuilt.
	var built time.Time
	if fi, err := os.Stat(pkgfile(pkg)); err == nil {
		built = fi.ModTime()
	}

	if built.IsZero() {
		return true
	}

	olderThan := func(file string) bool {
		fi, err := os.Stat(file)
		return err != nil || fi.ModTime().After(built)
	}

	// As a courtesy to developers installing new versions of the compiler
	// frequently, define that packages are stale if they are
	// older than the compiler, and commands if they are older than
	// the linker.  This heuristic will not work if the binaries are
	// back-dated, as some binary distributions may do, but it does handle
	// a very common case.
	if olderThan(pkg.tc.compiler()) {
		return true
	}
	if pkg.IsCommand() && olderThan(pkg.tc.linker()) {
		return true
	}

	// Package is stale if a dependency is newer.
	for _, p := range pkg.Imports() {
		if olderThan(pkgfile(p)) {
			return true
		}
	}

	srcs := stringList(pkg.GoFiles, pkg.CFiles, pkg.CXXFiles, pkg.MFiles, pkg.HFiles, pkg.SFiles, pkg.CgoFiles, pkg.SysoFiles, pkg.SwigFiles, pkg.SwigCXXFiles)

	// TODO(dfc) is pkg.isMain(), check to see if the binfile is up to date

	for _, src := range srcs {
		if olderThan(filepath.Join(pkg.Dir, src)) {
			return true
		}
	}

	return false
}

func stringList(args ...[]string) []string {
	var l []string
	for _, arg := range args {
		l = append(l, arg...)
	}
	return l
}
