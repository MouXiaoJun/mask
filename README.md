# go-mask

[中文](README_zh.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/MouXiaoJun/mask.svg)](https://pkg.go.dev/github.com/MouXiaoJun/mask)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg?style=flat-square)](LICENSE)

A **zero-dependency** data masking library for Go — struct-tag driven, in-place, with built-in formats for CN-sensitive fields and nested recursion.

## Quick start

```go
type User struct {
	Name     string `mask:"name"`
	Phone    string `mask:"phone"`
	IDCard   string `mask:"idcard"`
	Email    string `mask:"email"`
	Password string `mask:"password"`
}

user := User{Name: "张三", Phone: "13800138000", IDCard: "110101199003071234",
	Email: "zhangsan@example.com", Password: "secret"}
mask.Mask(&user)
// user.Phone == "138****8000", user.Name == "张*"
```

## Tag syntax

| Tag | Behavior |
| --- | --- |
| `mask:"phone"` | use a built-in / registered format |
| `mask:"3,4"` | keep first 3 + last 4 chars, mask the middle |
| `mask:"*"` | mask everything (same length) |
| `mask:"-"` | skip this field |
| none | not masked; nested structs still recursed |

## Built-in formats

`phone` (138\*\*\*\*8000), `idcard` (keep 6+4), `bankcard` (keep 4+4), `email` (keep first char of local part), `name` (keep first char), `address` (keep first 6), `password` / `*` (mask all).

All masking is **rune-safe** — Chinese names and addresses are never cut mid-character.

## Nested & slices

Structs, struct pointers, struct slices are recursed automatically; `Mask(&users)` handles a whole batch in one call.

## Custom formats

```go
mask.RegisterMask("carplate", func(s string) string {
	r := []rune(s)
	if len(r) < 4 { return s }
	return string(r[:2]) + "***" + string(r[len(r)-2:])
})
```

## Performance

Apple M5: simple struct (6 masked fields) **452 ns/op · 9 allocs**; 100-element slice **42 µs/op**. Field configs cached per type.

## Install

```bash
go get github.com/MouXiaoJun/mask
```

## License

MulanPSL-2.0
