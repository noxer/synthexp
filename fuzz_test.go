package synthexp

import (
	"math/rand/v2"
	"regexp"
	"regexp/syntax"
	"testing"
)

// fuzzSeeds are shared by the fuzz targets as a starting corpus.
var fuzzSeeds = []string{
	`hello`, `(?i)hello`, `[a-z]+`, `[^a]`, `\w\W\d\D\s\S`, `\p{Greek}`, `.`, `(?s).`,
	`a*`, `a+`, `a?`, `a{3}`, `a{2,5}`, `a{2,}`, `(ab|cd)*`, `(foo|bar)(baz)?`,
	`^abc$`, `\Aabc\z`, `(?m)^a$\n^b$`, `\bfoo\b`, `\w*\Bx`, `a^b`, `[^\x00-\x{10FFFF}]`,
	`(?P<name>\w+)@(\w+)\.com`,
}

// maxFuzzExpr bounds the expression length so nested repetitions can't make a
// single input produce enormous output.
const maxFuzzExpr = 64

// hasAssertion reports whether re contains an assertion that synthexp only
// satisfies on a best-effort basis.
func hasAssertion(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpBeginLine, syntax.OpEndLine, syntax.OpBeginText, syntax.OpEndText,
		syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		return true
	}
	for _, sub := range re.Sub {
		if hasAssertion(sub) {
			return true
		}
	}
	return false
}

// FuzzSynth checks that synthesis never panics and that, for expressions
// without assertions, every synthesized string matches the expression.
func FuzzSynth(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s, uint64(1))
	}
	f.Fuzz(func(t *testing.T, expr string, seed uint64) {
		if len(expr) > maxFuzzExpr {
			t.Skip()
		}
		syn, err := Compile(expr, WithMaxRepeat(3), WithRand(rand.New(rand.NewPCG(seed, seed))))
		if err != nil {
			t.Skip()
		}
		re, err := regexp.Compile(`^(?:` + expr + `)$`)
		if err != nil {
			t.Skip()
		}

		out := syn.Synth()
		if out == nil || hasAssertion(syn.re) {
			// Unsatisfiable expressions yield nil; assertions are best effort.
			return
		}
		if s := string(out); !re.MatchString(s) {
			t.Fatalf("%q synthesized %q which does not match", expr, s)
		}
	})
}

// FuzzCaptures checks that arbitrary capture overrides never cause a panic
// and are inserted verbatim when the expression is a single capture group.
func FuzzCaptures(f *testing.F) {
	for _, s := range fuzzSeeds {
		f.Add(s, "override")
	}
	f.Fuzz(func(t *testing.T, expr, capture string) {
		if len(expr) > maxFuzzExpr {
			t.Skip()
		}
		syn, err := Compile(expr, WithMaxRepeat(3))
		if err != nil {
			t.Skip()
		}
		syn.SynthString(Str(capture), nil, Str(capture))
		syn.SynthBytes([]byte(capture), nil)
		syn.Synth([]rune(capture))

		// Wrapping can fail to parse, e.g. an unterminated \Q swallows the ")".
		whole, err := Compile(`(` + expr + `)`)
		if err != nil {
			return
		}
		if got, want := whole.SynthString(Str(capture)), string([]rune(capture)); got != want {
			t.Fatalf("override %q of whole expression %q gave %q", capture, expr, got)
		}
	})
}
