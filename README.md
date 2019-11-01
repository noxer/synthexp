# synthexp
This package helps you generate strings that match your regular expression.

## installation
```bash
go get -u github.com/noxer/synthexp
```

## api
Synthesizing the string is a two step process, first you need to compile the regex.
```go
syn, err := synthexp.Compile("Hello (World|Earth|the (dear|awesome) User)\\. Here is some randomness [\\w]{3,8}")
if err != nil {
    fmt.Printf("Could not compile: %s\n", err)
}
```
Now you can use `syn` to generate as many matching strings as you want...
```bash
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

It can be useful for testing to have control over the captures in a regex. This can be provided by passing `*string`'s to the method (or `[]byte`/`[]rune` for `SynthBytes` and `Synth`). To skip captures and have the library generate random values you can pass `nil`. The provided values doesn't need to match the regex.
```go
str := syn.SynthString(synthexp.Str("Terra"))
fmt.Println(str)
```
```
Hello Terra. Here is some randomness Xf1
Hello Terra. Here is some randomness aC_FwmW
Hello Terra. Here is some randomness WX0
Hello Terra. Here is some randomness Qyp
Hello Terra. Here is some randomness Inf1y
Hello Terra. Here is some randomness 8G0oG
Hello Terra. Here is some randomness fIL
Hello Terra. Here is some randomness VKZ
Hello Terra. Here is some randomness 24d
Hello Terra. Here is some randomness lLN
```
```go
str := syn.SynthString(nil, synthexp.Str("glorious"))
fmt.Println(str)
```
```
Hello World. Here is some randomness Y55
Hello Earth. Here is some randomness dej
Hello World. Here is some randomness dpKI
Hello glorious User. Here is some randomness qF8g
Hello Earth. Here is some randomness 1Ucr
Hello glorious User. Here is some randomness FmhfX_
Hello glorious User. Here is some randomness KVcyhG
Hello World. Here is some randomness nX6a
Hello World. Here is some randomness 8DB
Hello World. Here is some randomness p3y1
```

## limits
The library currently can't reliably generate strings for regular expressions containing `^`, `$`, `\b` and `\B`.
