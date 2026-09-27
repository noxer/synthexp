package synthexp

import (
	"math/rand/v2"
	"regexp"
	"slices"
	"sync"
	"testing"
	"unicode/utf8"
)

const samples = 300

func seeded() Option {
	return WithRand(rand.New(rand.NewPCG(1, 2)))
}

func TestSynthMatches(t *testing.T) {
	exprs := []string{
		`hello`,
		`(?i)hello world`,
		`[a-z]`, `[^a]`, `[^\n]`, `\w`, `\W`, `\d`, `\D`, `\s`, `\S`,
		`\p{Greek}+`, `[\x{1F600}-\x{1F64F}]`, `[^\x00-\x7f]`,
		`.`, `(?s).`, `.*`,
		`a*`, `a+`, `a?`, `a{3}`, `a{2,5}`, `a{2,}`, `a{0}`, `(ab|cd)*`, `(?:x|yz)+`,
		`a|b|c`, `(foo|bar)(baz)?`, `((a|b)(c|d))+`,
		`^abc$`, `\Aabc\z`, `(?m)^a$\n^b$`, `(^a|b)c`, `(a|^)b`,
		`\bfoo\b`, `\w+\b \b\w+`, `\w*\Bx`, `x\B\w*`,
		`[\w.+-]+@[a-z0-9-]+\.[a-z]{2,6}`,
		`Hello (World|Earth|the (dear|awesome) User)\. Here is some randomness [\w]{3,8}`,
	}
	for _, expr := range exprs {
		t.Run(expr, func(t *testing.T) {
			syn := MustCompile(expr, seeded())
			re := regexp.MustCompile(`^(?:` + expr + `)$`)
			for range samples {
				s := syn.SynthString()
				if !re.MatchString(s) {
					t.Fatalf("synthesized %q which does not match", s)
				}
				if !utf8.ValidString(s) {
					t.Fatalf("synthesized invalid UTF-8 %q", s)
				}
			}
		})
	}
}

func TestNoMatch(t *testing.T) {
	for _, expr := range []string{`[^\x00-\x{10FFFF}]`, `a^b`, `a\Ab`, `(a|b)^c`} {
		if got := MustCompile(expr).Synth(); got != nil {
			t.Errorf("%s: got %q, want nil", expr, string(got))
		}
	}
}

func TestRepeatBounds(t *testing.T) {
	tests := []struct {
		expr     string
		opts     []Option
		min, max int
	}{
		{`a*`, nil, 0, DefaultMaxRepeat},
		{`a+`, nil, 1, DefaultMaxRepeat + 1},
		{`a?`, nil, 0, 1},
		{`a{4}`, nil, 4, 4},
		{`a{2,5}`, nil, 2, 5},
		{`a{3,}`, []Option{WithMaxRepeat(2)}, 3, 5},
		{`a*`, []Option{WithMaxRepeat(0)}, 0, 0},
	}
	for _, tt := range tests {
		syn := MustCompile(tt.expr, append(tt.opts, seeded())...)
		seen := map[int]bool{}
		for range 2000 {
			seen[len(syn.Synth())] = true
		}
		for n := range seen {
			if n < tt.min || n > tt.max {
				t.Errorf("%s: length %d outside [%d, %d]", tt.expr, n, tt.min, tt.max)
			}
		}
		if !seen[tt.min] || !seen[tt.max] {
			t.Errorf("%s: expected both bounds %d and %d to occur, got %v", tt.expr, tt.min, tt.max, seen)
		}
	}
}

func TestCaptures(t *testing.T) {
	syn := MustCompile(`Hello (World|Earth|the (dear|awesome) User)`, seeded())

	if got := syn.SynthString(Str("Terra")); got != "Hello Terra" {
		t.Errorf("got %q", got)
	}
	if got := syn.SynthString(Str("")); got != "Hello " {
		t.Errorf("empty override: got %q", got)
	}
	if got := string(syn.SynthBytes([]byte("Terra"))); got != "Hello Terra" {
		t.Errorf("bytes: got %q", got)
	}
	if got := string(syn.Synth([]rune("Terra"))); got != "Hello Terra" {
		t.Errorf("runes: got %q", got)
	}

	// nil skips a capture; overrides beyond the number of captures are ignored.
	allowed := []string{"Hello World", "Hello Earth", "Hello the glorious User"}
	for range samples {
		for _, got := range []string{
			syn.SynthString(nil, Str("glorious"), Str("ignored")),
			string(syn.SynthBytes(nil, []byte("glorious"))),
		} {
			if !slices.Contains(allowed, got) {
				t.Fatalf("got %q", got)
			}
		}
	}
}

func TestDeterministic(t *testing.T) {
	gen := func() []string {
		syn := MustCompile(`[a-z]{5,10}|\d+`, WithRand(rand.New(rand.NewPCG(42, 42))))
		var out []string
		for range 20 {
			out = append(out, syn.SynthString())
		}
		return out
	}
	if a, b := gen(), gen(); !slices.Equal(a, b) {
		t.Errorf("same seed produced different output:\n%q\n%q", a, b)
	}
}

func TestConcurrent(t *testing.T) {
	syn := MustCompile(`\w+@\w+\.com`)
	re := regexp.MustCompile(`^\w+@\w+\.com$`)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				if s := syn.SynthString(); !re.MatchString(s) {
					t.Errorf("no match: %q", s)
				}
			}
		}()
	}
	wg.Wait()
}

func TestCompileError(t *testing.T) {
	if _, err := Compile(`a(`); err == nil {
		t.Error("expected error")
	}
	defer func() {
		if recover() == nil {
			t.Error("expected MustCompile to panic")
		}
	}()
	MustCompile(`a(`)
}

func BenchmarkSynthString(b *testing.B) {
	syn := MustCompile(`Hello (World|Earth|the (dear|awesome) User)\. Here is some randomness [\w]{3,8}`)
	for range b.N {
		syn.SynthString()
	}
}
