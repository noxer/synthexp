# synthexp

[![Go Reference](https://pkg.go.dev/badge/github.com/noxer/synthexp.svg)](https://pkg.go.dev/github.com/noxer/synthexp)
[![CI](https://github.com/noxer/synthexp/actions/workflows/ci.yml/badge.svg)](https://github.com/noxer/synthexp/actions/workflows/ci.yml)

This package helps you generate strings that match your regular expression. It
understands the same syntax as Go's [`regexp`](https://pkg.go.dev/regexp) package.

## installation
```bash
go get github.com/noxer/synthexp
```
Requires Go 1.22 or newer.

## api
Synthesizing the string is a two step process, first you need to compile the regex.
```go
syn, err := synthexp.Compile(`Hello (World|Earth|the (dear|awesome) User)\. Here is some randomness [\w]{3,8}`)
if err != nil {
    fmt.Printf("Could not compile: %s\n", err)
}
```
Or, for expressions known to be valid, `syn := synthexp.MustCompile(...)`.

Now you can use `syn` to generate as many matching strings as you want...
```go
str := syn.SynthString()
fmt.Println(str)
```
Printing those strings gave me the following output.
```
Hello dear User. Here is some randomness O1iIJ
Hello Earth. Here is some randomness rj3vR
Hello World. Here is some randomness SqO
Hello dear User. Here is some randomness fvM
Hello World. Here is some randomness eHIdQn
Hello World. Here is some randomness tb8
Hello Earth. Here is some randomness xzaD
Hello awesome User. Here is some randomness HNU
Hello World. Here is some randomness qr2HN3S
Hello Earth. Here is some randomness oKL
```

### fixing captures
It can be useful for testing to have control over the captures in a regex. This can be provided by passing `*string`'s to the method (or `[]byte`/`[]rune` for `SynthBytes` and `Synth`). To skip captures and have the library generate random values you can pass `nil`. The provided values don't need to match the regex.
```go
str := syn.SynthString(synthexp.Str("Terra"))
fmt.Println(str)
```
```
Hello Terra. Here is some randomness Xf1
Hello Terra. Here is some randomness aC_FwmW
Hello Terra. Here is some randomness WX0
```
```go
str := syn.SynthString(nil, synthexp.Str("glorious"))
fmt.Println(str)
```
```
Hello World. Here is some randomness Y55
Hello glorious User. Here is some randomness qF8g
Hello Earth. Here is some randomness 1Ucr
```

### options
`Compile` and `MustCompile` accept options:

- `synthexp.WithRand(r *rand.Rand)` draws randomness from `r` (`math/rand/v2`), e.g. for reproducible output with a fixed seed. Without it, the auto-seeded global source is used.
- `synthexp.WithMaxRepeat(n)` limits how many repetitions `*`, `+` and `{n,}` may add beyond their minimum (default `32`).

```go
syn := synthexp.MustCompile(`[a-z]+`, synthexp.WithRand(rand.New(rand.NewPCG(1, 2))), synthexp.WithMaxRepeat(8))
```

A `*Synthexp` is safe for concurrent use, unless it was created with `WithRand` (a `*rand.Rand` is not safe for concurrent use).

## limits
- Anchors and word boundaries (`^`, `$`, `\A`, `\z`, `\b`, `\B`) are supported by retrying generation until they hold. For expressions that can only rarely satisfy them this is best effort, and a non-matching string may be returned.
- Expressions that can never match anything (e.g. `a^b`) produce `nil`.
- Character classes prefer printable ASCII when they contain any; `.` draws from the exported `Alphabet`.
