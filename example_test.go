package synthexp_test

import (
	"fmt"
	"math/rand/v2"

	"github.com/noxer/synthexp"
)

func Example() {
	syn, err := synthexp.Compile(`Hello (World|Earth|the (dear|awesome) User)\. Here is some randomness \w{3,8}`)
	if err != nil {
		fmt.Printf("Could not compile: %s\n", err)
		return
	}
	fmt.Println(len(syn.SynthString()) > 0)
	// Output: true
}

func ExampleSynthexp_SynthString() {
	syn := synthexp.MustCompile(`Hello (World|Earth)!`)

	// Fix the first capture; pass nil to let synthexp fill a capture itself.
	fmt.Println(syn.SynthString(new("Terra")))
	// Output: Hello Terra!
}

func ExampleWithRand() {
	seed := func() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }
	a := synthexp.MustCompile(`[a-z]{8}`, synthexp.WithRand(seed()))
	b := synthexp.MustCompile(`[a-z]{8}`, synthexp.WithRand(seed()))

	// The same seed yields the same strings.
	fmt.Println(a.SynthString() == b.SynthString())
	// Output: true
}
