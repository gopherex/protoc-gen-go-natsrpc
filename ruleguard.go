// Canonical gocritic ruleguard rules for Go repos. Copy as ruleguard.go next to .golangci.yaml
// (the module root); never edit the copy. The build tag keeps it out of every build.

//go:build ruleguard

package rules

import (
	"strings"

	"github.com/quasilyte/go-ruleguard/dsl"
	"github.com/quasilyte/go-ruleguard/dsl/types"
)

// ADAPT: generated contract package root (import path) and generator fixture packages (regexp).
const (
	generatedContractPackage  = "example.invalid/module/pkg/gen"
	generatedContractFixtures = `$^`
)

// mustOnlyAtPackageLevel: a Must* helper panics on error. It is allowed only as the direct
// initializer of a package-level variable, where a bad value stops the process at start-up;
// inside a function the error is handled as a value. Known gap: a grouped `var (...)` inside a
// function is not matched (gogrep cannot look into a group), review catches it.
func mustOnlyAtPackageLevel(m dsl.Matcher) {
	m.Match(`$f($*_)`).
		Where(m["f"].Text.Matches(`(^|\.)Must[A-Z0-9]?\w*$`) && !m["$$"].Node.Parent().Is(`ValueSpec`)).
		Report(`$f panics on error: call it only as a package-level var initializer, else handle the error`)

	m.Match(`var $_ = $f($*_)`, `var $_ $_ = $f($*_)`).
		Where(m["f"].Text.Matches(`(^|\.)Must[A-Z0-9]?\w*$`) && !m["$$"].Node.Parent().Is(`File`)).
		Report(`$f panics on error: a local var is not start-up; handle the error`)
}

// generatedContractLiteral keeps message construction on the generated builder's typed steps.
func generatedContractLiteral(m dsl.Matcher) {
	m.Match(`$_{$*_}`, `{$*_,}`).
		Where(!m.File().PkgPath.Matches(`^`+generatedContractPackage+`(/|$)`) &&
			!m.File().PkgPath.Matches(generatedContractFixtures) && m["$$"].Filter(isGeneratedContractStruct)).
		Report(`use the generated contract builder New<Msg>()…Build() instead of a composite literal`)
}

func isGeneratedContractStruct(ctx *dsl.VarFilterContext) bool {
	typ := ctx.Type
	ptr := types.AsPointer(typ)
	if ptr != nil {
		typ = ptr.Elem()
	}

	// ID values have an array representation; the corresponding wire ID is a message.
	if types.AsStruct(typ.Underlying()) == nil {
		return false
	}

	name := typ.String()

	return strings.HasPrefix(name, generatedContractPackage+"/") || strings.HasPrefix(name, generatedContractPackage+".")
}
