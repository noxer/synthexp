// Package synthexp generates random strings that match a regular expression.
//
// Expressions use the same syntax as the standard library's regexp package
// (RE2 / Perl flavour):
//
//	syn := synthexp.MustCompile(`Hello (World|Earth)\. [a-z]{3,8}`)
//	fmt.Println(syn.SynthString())
//
// A *Synthexp is safe for concurrent use unless it was created with WithRand,
// because *rand.Rand itself is not safe for concurrent use.
package synthexp

import (
	"math/rand/v2"
	"regexp/syntax"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var (
	// WordRunes defines the characters that can be used in words (as defined by regex).
	WordRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_")
	// NonWordRunes defines the characters that can't be used in words (as defined by regex).
	NonWordRunes = []rune(" ,.-;:!\"§$%&\\/()=?`´#'+*}][{\n")
	// Alphabet defines the characters used to synthesize the wildcard "."
	// (a newline is only produced when the s flag is set).
	Alphabet = append(append([]rune{}, WordRunes...), NonWordRunes...)
)

// DefaultMaxRepeat is the default upper bound for unbounded repetitions (*, + and {n,}).
const DefaultMaxRepeat = 32

// maxAttempts bounds how often synthesis is retried when a generated string
// violates an assertion such as ^, $, \b or \B.
const maxAttempts = 100

// Synthexp offers functionality to generate strings from a regex.
type Synthexp struct {
	re        *syntax.Regexp
	rnd       *rand.Rand
	maxRepeat int
}

// Option configures a Synthexp.
type Option func(*Synthexp)

// WithRand makes the Synthexp draw its randomness from r, e.g. to get
// reproducible output with a fixed seed. A *rand.Rand is not safe for
// concurrent use, so neither is a Synthexp created with this option.
func WithRand(r *rand.Rand) Option {
	return func(se *Synthexp) { se.rnd = r }
}

// WithMaxRepeat sets how many repetitions an unbounded repetition (*, + and
// {n,}) may add beyond its minimum. Negative values are treated as zero.
func WithMaxRepeat(n int) Option {
	return func(se *Synthexp) { se.maxRepeat = max(n, 0) }
}

// Compile parses the expression and prepares synthesis.
func Compile(expr string, opts ...Option) (*Synthexp, error) {
	re, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return nil, err
	}
	se := &Synthexp{
		re:        re,
		maxRepeat: DefaultMaxRepeat,
	}
	for _, opt := range opts {
		opt(se)
	}
	return se, nil
}

// MustCompile is like Compile but panics if the expression cannot be parsed.
func MustCompile(expr string, opts ...Option) *Synthexp {
	se, err := Compile(expr, opts...)
	if err != nil {
		panic(`synthexp: Compile(` + quote(expr) + `): ` + err.Error())
	}
	return se
}

// SynthString synthesizes a random string that is matched by expr. The parameters in caps allow you to provide fixed values for captures within the regexp. They are not checked against the expression which can lead to non-matching strings. You can provide nil for captures that should be filled by synthexp.
func (se *Synthexp) SynthString(caps ...*string) string {
	runeCaps := make([][]rune, len(caps))
	for i, c := range caps {
		if c != nil {
			runeCaps[i] = toRunes(*c)
		}
	}
	return string(se.Synth(runeCaps...))
}

// SynthBytes synthesizes a random string that is matched by expr. The parameters in caps allow you to provide fixed values for captures within the regexp. They are not checked against the expression which can lead to non-matching strings. You can provide nil for captures that should be filled by synthexp.
func (se *Synthexp) SynthBytes(caps ...[]byte) []byte {
	runeCaps := make([][]rune, len(caps))
	for i, c := range caps {
		if c != nil {
			runeCaps[i] = toRunes(string(c))
		}
	}
	return []byte(string(se.Synth(runeCaps...)))
}

// Synth synthesizes a random []rune that is matched by expr. The parameters in caps allow you to provide fixed values for captures within the regexp. They are not checked against the expression which can lead to non-matching strings. You can provide nil for captures that should be filled by synthexp.
//
// Synth returns nil if the expression cannot match anything (e.g. `a^b`).
// Assertions (^, $, \b, \B) are satisfied by retrying; if no attempt satisfies
// them, the last attempt is returned even though it may not match.
func (se *Synthexp) Synth(caps ...[]rune) []rune {
	g := generator{
		rnd:       se.rnd,
		maxRepeat: se.maxRepeat,
		caps:      caps,
	}
	var best []rune
	for range maxAttempts {
		g.out, g.asserts = g.out[:0], g.asserts[:0]
		if !g.gen(se.re) {
			continue
		}
		best = append(best[:0], g.out...)
		if g.assertionsHold() {
			break
		}
	}
	return best
}

// toRunes converts s to a non-nil []rune, so that an empty override is not
// mistaken for "generate this capture".
func toRunes(s string) []rune {
	return append([]rune{}, []rune(s)...)
}

// Str returns a pointer to the string.
func Str(str string) *string {
	return &str
}

// assertion is a zero-width assertion that can only be checked once the
// characters after it have been generated.
type assertion struct {
	op  syntax.Op
	pos int
}

// generator holds the state of a single synthesis run.
type generator struct {
	rnd       *rand.Rand
	maxRepeat int
	caps      [][]rune
	out       []rune
	asserts   []assertion
}

func (g *generator) intN(n int) int {
	if g.rnd != nil {
		return g.rnd.IntN(n)
	}
	return rand.IntN(n)
}

// gen appends a string matched by re to g.out. It returns false if re cannot
// match at the current position, in which case the caller is responsible for
// restoring g.out and g.asserts.
func (g *generator) gen(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpNoMatch:
		return false
	case syntax.OpEmptyMatch:
		return true
	case syntax.OpLiteral:
		for _, r := range re.Rune {
			if re.Flags&syntax.FoldCase != 0 {
				r = g.foldCase(r)
			}
			g.out = append(g.out, r)
		}
		return true
	case syntax.OpCharClass:
		r, ok := g.pickClass(re.Rune)
		if ok {
			g.out = append(g.out, r)
		}
		return ok
	case syntax.OpAnyCharNotNL:
		return g.pickAlphabet(false)
	case syntax.OpAnyChar:
		return g.pickAlphabet(true)
	case syntax.OpBeginText:
		return len(g.out) == 0
	case syntax.OpBeginLine:
		return len(g.out) == 0 || g.out[len(g.out)-1] == '\n'
	case syntax.OpEndText, syntax.OpEndLine, syntax.OpWordBoundary, syntax.OpNoWordBoundary:
		g.asserts = append(g.asserts, assertion{op: re.Op, pos: len(g.out)})
		return true
	case syntax.OpCapture:
		if re.Cap <= len(g.caps) && g.caps[re.Cap-1] != nil {
			g.out = append(g.out, g.caps[re.Cap-1]...)
			return true
		}
		return g.gen(re.Sub[0])
	case syntax.OpStar:
		return g.repeat(re.Sub[0], 0, -1)
	case syntax.OpPlus:
		return g.repeat(re.Sub[0], 1, -1)
	case syntax.OpQuest:
		return g.repeat(re.Sub[0], 0, 1)
	case syntax.OpRepeat:
		return g.repeat(re.Sub[0], re.Min, re.Max)
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if !g.gen(sub) {
				return false
			}
		}
		return true
	case syntax.OpAlternate:
		// Start at a random branch and fall through to the others if it can't match.
		outLen, assertLen := len(g.out), len(g.asserts)
		start := g.intN(len(re.Sub))
		for i := range re.Sub {
			if g.gen(re.Sub[(start+i)%len(re.Sub)]) {
				return true
			}
			g.out, g.asserts = g.out[:outLen], g.asserts[:assertLen]
		}
		return false
	}
	return false
}

// repeat generates between lo and hi repetitions of re; hi == -1 means unbounded.
func (g *generator) repeat(re *syntax.Regexp, lo, hi int) bool {
	if hi < 0 {
		hi = lo + g.maxRepeat
	}
	n := lo + g.intN(max(hi, lo)-lo+1)
	for range n {
		if !g.gen(re) {
			return false
		}
	}
	return true
}

// foldCase returns a random rune from the case folding orbit of r.
func (g *generator) foldCase(r rune) rune {
	orbit := []rune{r}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		orbit = append(orbit, f)
	}
	return orbit[g.intN(len(orbit))]
}

func (g *generator) pickAlphabet(withNL bool) bool {
	alphabet := Alphabet
	if !withNL {
		alphabet = make([]rune, 0, len(Alphabet))
		for _, r := range Alphabet {
			if r != '\n' {
				alphabet = append(alphabet, r)
			}
		}
	}
	if len(alphabet) == 0 {
		// Fall back to printable ASCII if the alphabet was emptied by the user.
		r, _ := g.pickClass([]rune{' ', '~'})
		g.out = append(g.out, r)
		return true
	}
	g.out = append(g.out, alphabet[g.intN(len(alphabet))])
	return true
}

// preferred lists the rune ranges that are picked from a character class when
// the class contains any of them, so that e.g. [^a] or \W don't produce
// control characters or obscure Unicode.
var preferred = []rune{'\t', '\n', ' ', '~'}

// pickClass picks a random rune from the class given as sorted pairs of
// inclusive ranges. It returns false for an empty class.
func (g *generator) pickClass(ranges []rune) (rune, bool) {
	if r, ok := g.pickRange(intersect(ranges, preferred)); ok {
		return r, true
	}
	// Try a few times to pick something printable before giving up on it.
	var r rune
	var ok bool
	for range 8 {
		if r, ok = g.pickRange(ranges); !ok || (unicode.IsGraphic(r) && utf8.ValidRune(r)) {
			break
		}
	}
	return r, ok
}

// pickRange picks a uniformly random rune from the given ranges.
func (g *generator) pickRange(ranges []rune) (rune, bool) {
	total := 0
	for i := 0; i+1 < len(ranges); i += 2 {
		total += int(ranges[i+1]-ranges[i]) + 1
	}
	if total == 0 {
		return 0, false
	}
	n := g.intN(total)
	for i := 0; i+1 < len(ranges); i += 2 {
		size := int(ranges[i+1]-ranges[i]) + 1
		if n < size {
			return ranges[i] + rune(n), true
		}
		n -= size
	}
	panic("unreachable")
}

// intersect returns the intersection of two sorted range lists.
func intersect(a, b []rune) []rune {
	var res []rune
	for i := 0; i+1 < len(a); i += 2 {
		for j := 0; j+1 < len(b); j += 2 {
			lo, hi := max(a[i], b[j]), min(a[i+1], b[j+1])
			if lo <= hi {
				res = append(res, lo, hi)
			}
		}
	}
	return res
}

// assertionsHold reports whether all deferred assertions are satisfied by g.out.
func (g *generator) assertionsHold() bool {
	for _, a := range g.asserts {
		var before, after rune = -1, -1
		if a.pos > 0 {
			before = g.out[a.pos-1]
		}
		if a.pos < len(g.out) {
			after = g.out[a.pos]
		}
		var ok bool
		switch a.op {
		case syntax.OpEndText:
			ok = after == -1
		case syntax.OpEndLine:
			ok = after == -1 || after == '\n'
		case syntax.OpWordBoundary:
			ok = syntax.IsWordChar(before) != syntax.IsWordChar(after)
		case syntax.OpNoWordBoundary:
			ok = syntax.IsWordChar(before) == syntax.IsWordChar(after)
		}
		if !ok {
			return false
		}
	}
	return true
}

func quote(s string) string {
	if strconv.CanBackquote(s) {
		return "`" + s + "`"
	}
	return strconv.Quote(s)
}
