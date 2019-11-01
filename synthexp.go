package synthexp

import (
	"math/rand"
	"regexp/syntax"
	"unicode"
)

var (
	// WordRunes defines the characters that can be used in words (as defined by regex).
	WordRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	// NonWordRunes defines the characters that can't be used in words (as defined by regex).
	NonWordRunes = []rune(" ,.-;:_!\"§$%&\\/()=?`´#'+*}][{\n")
	// Alphabet defines the characters used for arbitrary characters, must end with a newline.
	Alphabet = append(WordRunes, NonWordRunes...)
)

// Synthexp offers functionality to generate strings from a regex.
type Synthexp struct {
	re *syntax.Regexp
}

// Compile parses the expression and prepared synthesis.
func Compile(expr string) (*Synthexp, error) {
	re, err := syntax.Parse(expr, syntax.Perl)
	if err != nil {
		return nil, err
	}
	re = re.Simplify()
	return &Synthexp{
		re: re,
	}, nil
}

// SynthString synthesizes a random string that is matched by expr. The parameters in caps allow you to provide fixed values for captures within the regexp. They are not checked against the expression which can lead to non-matching strings. You can provide nil for captures that should be filled by synthexp.
func (se *Synthexp) SynthString(caps ...*string) string {
	runeCaps := make([][]rune, len(caps))
	for i, c := range caps {
		if c == nil {
			continue
		}
		runeCaps[i] = []rune(*c)
	}
	return string(se.Synth(runeCaps...))
}

// SynthBytes synthesizes a random string that is matched by expr. The parameters in caps allow you to provide fixed values for captures within the regexp. They are not checked against the expression which can lead to non-matching strings. You can provide nil for captures that should be filled by synthexp.
func (se *Synthexp) SynthBytes(caps ...[]byte) []byte {
	runeCaps := make([][]rune, len(caps))
	for i, c := range caps {
		runeCaps[i] = []rune(string(c))
	}
	return []byte(string(se.Synth(runeCaps...)))
}

// Synth synthesises a random []rune that is matched by expr. The parameters in caps allow you to provide fixed values for captures within the regexp. They are not checked against the expression which can lead to non-matching strings. You can provide nil for captures that should be filled by synthexp.
func (se *Synthexp) Synth(caps ...[]rune) []rune {
	r, _ := synth(0, se.re, caps)
	return r
}

func synth(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	switch re.Op {
	case syntax.OpNoMatch:
		return synthNoMatch(prev, re, caps)
	case syntax.OpEmptyMatch:
		return synthEmptyMatch(prev, re, caps)
	case syntax.OpLiteral:
		return synthLiteral(prev, re, caps)
	case syntax.OpCharClass:
		return synthCharClass(prev, re, caps)
	case syntax.OpAnyCharNotNL:
		return synthAnyCharNotNL(prev, re, caps)
	case syntax.OpAnyChar:
		return synthAnyChar(prev, re, caps)
	case syntax.OpBeginLine, syntax.OpBeginText:
		return synthBeginText(prev, re, caps)
	case syntax.OpCapture:
		return synthCapture(prev, re, caps)
	case syntax.OpStar:
		return synthStar(prev, re, caps)
	case syntax.OpPlus:
		return synthStar(prev, re, caps)
	case syntax.OpQuest:
		return synthQuest(prev, re, caps)
	case syntax.OpRepeat:
		return synthRepeat(prev, re, caps)
	case syntax.OpConcat:
		return synthConcat(prev, re, caps)
	case syntax.OpAlternate:
		return synthAlternate(prev, re, caps)
	}
	return []rune{}, false
}

func synthNoMatch(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	return nil, false // indicate an invalid match
}

func synthEmptyMatch(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	return []rune{}, false // return an empty match
}

func synthLiteral(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	return re.Rune, false
}

func synthCharClass(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	runes := expandRanges(re.Rune)
	i := rand.Intn(len(runes))
	return []rune{runes[i]}, false
}

func synthAnyCharNotNL(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	i := rand.Intn(len(Alphabet) - 1)
	return []rune{Alphabet[i]}, false
}

func synthAnyChar(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	i := rand.Intn(len(Alphabet))
	return []rune{Alphabet[i]}, false
}

func synthBeginText(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	if prev == 0 {
		return []rune{}, false
	}
	return nil, false // there was a rune before this one, this can't work
}

func synthCapture(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	if re.Cap <= len(caps) && caps[re.Cap-1] != nil {
		return caps[re.Cap-1], false
	}
	return synth(prev, re.Sub[0], caps)
}

func synthStar(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	n := rand.Intn(32)
	if n == 0 {
		return []rune{}, false
	}
	var res []rune
	for i := 0; i < n; i++ {
		if len(res) > 0 {
			prev = res[len(res)-1]
		}
		r, _ := synth(prev, re.Sub[0], caps)
		if r == nil {
			return nil, false
		}
		res = append(res, r...)
	}
	return res, false
}

func synthPlus(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	n := rand.Intn(31) + 1
	res := []rune{}
	for i := 0; i < n; i++ {
		if len(res) > 0 {
			prev = res[len(res)-1]
		}
		r, _ := synth(prev, re.Sub[0], caps)
		if r == nil {
			return nil, false
		}
		res = append(res, r...)
	}
	return res, false
}

func synthQuest(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	n := rand.Intn(2)
	res := []rune{}
	if n == 1 {
		r, _ := synth(prev, re.Sub[0], caps)
		if r == nil {
			return nil, false
		}
		res = append(res, r...)
	}
	return res, false
}

func synthRepeat(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	min := re.Min
	max := re.Max
	if max == -1 {
		max = 32
	}
	if max < min {
		max = min
	}

	n := rand.Intn(max-min) + min
	res := []rune{}
	for i := 0; i < n; i++ {
		if len(res) > 0 {
			prev = res[len(res)-1]
		}
		r, _ := synth(prev, re.Sub[0], caps)
		if r == nil {
			return nil, false
		}
		res = append(res, r...)
	}
	return res, false
}

func synthConcat(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	var res []rune
	for _, s := range re.Sub {
		if len(res) > 0 {
			prev = res[len(res)-1]
		}
		r, _ := synth(prev, s, caps)
		if r == nil {
			return nil, false
		}
		res = append(res, r...)
	}
	return res, false
}

func synthAlternate(prev rune, re *syntax.Regexp, caps [][]rune) ([]rune, bool) {
	i := rand.Intn(len(re.Sub))
	for range re.Sub {
		r, _ := synth(prev, re.Sub[i], caps)
		if r != nil {
			return r, false
		}
		i = (i + 1) % len(re.Sub)
	}
	return nil, false
}

func expandRanges(ranges []rune) []rune {
	var expanded []rune
	for i := 0; i < len(ranges)-1; i += 2 {
		expanded = append(expanded, expandRange(ranges[i], ranges[i+1])...)
	}
	return expanded
}

func expandRange(from, to rune) []rune {
	if to == unicode.MaxRune {
		to = 126
	}
	if to < from {
		to = from
	}
	expanded := make([]rune, to-from+1)
	for r := from; r <= to; r++ {
		expanded[r-from] = r
	}
	return expanded
}

// Str returns a pointer to the string.
func Str(str string) *string {
	return &str
}
