package interp

import (
	"go/build"
	"go/parser"
	"path"
	"strconv"
	"strings"
)

// buildOk returns true if a file or script matches build constraints
// as specified in https://golang.org/pkg/go/build/#hdr-Build_Constraints
func (interp *Interpreter) buildOk(ctx build.Context, name, src string) bool {
	// Extract comments before the first clause
	f, err := parser.ParseFile(interp.fset, name, src, parser.PackageClauseOnly|parser.ParseComments)
	if err != nil {
		return false
	}
	for _, g := range f.Comments {
		// in file, evaluate the AND of multiple line build constraints
		for _, line := range strings.Split(strings.TrimSpace(g.Text()), "\n") {
			if !buildLineOk(ctx, line) {
				return false
			}
		}
	}
	return true
}

// buildLineOk returns true if line is not a build constraint or
// if build constraint is satisfied
func buildLineOk(ctx build.Context, line string) (ok bool) {
	if len(line) < 7 || line[:7] != "+build " {
		return true
	}
	// In line, evaluate the OR of space-separated options
	options := strings.Split(strings.TrimSpace(line[6:]), " ")
	for _, o := range options {
		if ok = buildOptionOk(ctx, o); ok {
			break
		}
	}
	return ok
}

// buildOptionOk return true if all comma separated tags match, false otherwise
func buildOptionOk(ctx build.Context, tag string) bool {
	// in option, evaluate the AND of individual tags
	for _, t := range strings.Split(tag, ",") {
		if !buildTagOk(ctx, t) {
			return false
		}
	}
	return true
}

// buildTagOk returns true if a build tag matches, false otherwise
// if first character is !, result is negated
func buildTagOk(ctx build.Context, s string) (r bool) {
	not := s[0] == '!'
	if not {
		s = s[1:]
	}
	switch {
	case contains(ctx.BuildTags, s):
		r = true
	case s == ctx.GOOS:
		r = true
	case s == ctx.GOARCH:
		r = true
	case len(s) > 4 && s[:4] == "go1.":
		if n, err := strconv.Atoi(s[4:]); err != nil {
			r = false
		} else {
			r = goMinorVersion(ctx) >= n
		}
	}
	if not {
		r = !r
	}
	return
}

func contains(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// goMinorVersion returns the go minor version number
func goMinorVersion(ctx build.Context) int {
	current := ctx.ReleaseTags[len(ctx.ReleaseTags)-1]

	v := strings.Split(current, ".")
	if len(v) < 2 {
		panic("unsupported Go version: " + current)
	}

	m, err := strconv.Atoi(v[1])
	if err != nil {
		panic("unsupported Go version: " + current)
	}
	return m
}

// skipFile returns true if file should be skipped
func skipFile(ctx build.Context, p string) bool {
	if !strings.HasSuffix(p, ".go") {
		return true
	}
	p = strings.TrimSuffix(path.Base(p), ".go")
	if strings.HasSuffix(p, "_test") {
		return true
	}
	i := strings.Index(p, "_")
	if i < 0 {
		return false
	}
	a := strings.Split(p[i+1:], "_")
	last := len(a) - 1
	if last1 := last - 1; last1 >= 0 && a[last1] == ctx.GOOS && a[last] == ctx.GOARCH {
		return false
	}
	if s := a[last]; s != ctx.GOOS && s != ctx.GOARCH && knownOs[s] || knownArch[s] {
		return true
	}
	return false
}

var knownOs = map[string]bool{
	"aix":       true,
	"android":   true,
	"darwin":    true,
	"dragonfly": true,
	"freebsd":   true,
	"js":        true,
	"linux":     true,
	"nacl":      true,
	"netbsd":    true,
	"openbsd":   true,
	"plan9":     true,
	"solaris":   true,
	"windows":   true,
}

var knownArch = map[string]bool{
	"386":      true,
	"amd64":    true,
	"amd64p32": true,
	"arm":      true,
	"arm64":    true,
	"mips":     true,
	"mips64":   true,
	"mips64le": true,
	"mipsle":   true,
	"ppc64":    true,
	"ppc64le":  true,
	"s390x":    true,
	"wasm":     true,
}
