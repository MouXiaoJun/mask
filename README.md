# go-mask

[中文](README_zh.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/MouXiaoJun/mask.svg)](https://pkg.go.dev/github.com/MouXiaoJun/mask)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg?style=flat-square)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/MouXiaoJun/mask.svg?style=flat-square)](https://github.com/MouXiaoJun/mask/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MouXiaoJun/mask?style=flat-square)](https://goreportcard.com/report/github.com/MouXiaoJun/mask)

A **zero-dependency** data masking library for Go: struct-tag driven, in-place, with built-in formats for the most common sensitive fields and automatic nested recursion.

> 🎯 **Declarative masking** — `mask:"phone"` or `mask:"3,4"`; the format name *is* the tag
>
> ⚡ **In-place** — `Mask(&user)` rewrites field values; no new objects, no return value
>
> 🌏 **Rune-safe** — Chinese names, addresses and emoji are never cut mid-character
>
> 🧩 **Family** — tags coexist with the author's `validator`, `dict_trans` and `copier` libraries: validate first, mask last

## Features

- ✅ **Zero dependencies** — stdlib only (`reflect` / `sync` / `sync/atomic`)
- ✅ **Tag driven** — `mask:"phone"` masks a field in one line; rules live next to the data
- ✅ **In-place** — `Mask(&user)` mutates the passed struct directly
- ✅ **8 built-in formats** — phone / idcard / bankcard / email / name / address / password / wildcard
- ✅ **Generic keep format** — `mask:"3,4"` keeps the first 3 and last 4 chars, any n/m
- ✅ **Nested recursion** — structs, struct pointers and struct slices are recursed automatically
- ✅ **Top-level batches** — `Mask(&users)` masks a whole slice in one call
- ✅ **Custom formats** — `RegisterMask("carplate", fn)` takes effect immediately, including on already-masked types
- ✅ **Rune-safe** — all formats operate on Unicode code points
- ✅ **Per-type config cache** — zero tag parsing and zero format lookup after the first call; lock-free reads
- ✅ **Generic entry point** — `MaskOf(&user)` guarantees `*T` at compile time
- ✅ **Visible errors** — a misspelled format name fails on the first call instead of silently leaking; `ErrNotPointer` / `ErrNotStruct`

## Install

Requires Go 1.21+:

```bash
go get github.com/MouXiaoJun/mask
```

## Quick start

```go
package main

import (
	"fmt"

	"github.com/MouXiaoJun/mask"
)

type User struct {
	Name     string `mask:"name"`
	Phone    string `mask:"phone"`
	IDCard   string `mask:"idcard"`
	Email    string `mask:"email"`
	Password string `mask:"password"`
	Note     string // no tag: not masked
}

func main() {
	user := User{
		Name:     "张三",
		Phone:    "13800138000",
		IDCard:   "110101199003071234",
		Email:    "zhangsan@example.com",
		Password: "secret123",
		Note:     "untouched",
	}

	if err := mask.Mask(&user); err != nil {
		panic(err)
	}

	fmt.Println(user.Name)     // 张*
	fmt.Println(user.Phone)    // 138****8000
	fmt.Println(user.IDCard)   // 110101********1234
	fmt.Println(user.Email)    // z*******@example.com
	fmt.Println(user.Password) // *********
	fmt.Println(user.Note)     // untouched
}
```

Generic variant, `MaskOf[T](v *T)` — the pointer requirement is enforced at compile time:

```go
if err := mask.MaskOf(&user); err != nil {
	panic(err)
}
```

Errors are comparable with `errors.Is`:

| Error | When |
| --- | --- |
| `mask.ErrNotPointer` | argument is not a non-nil pointer (value, `nil`, or `(*T)(nil)`) |
| `mask.ErrNotStruct` | pointer target is not a struct or a struct slice (e.g. `*int`) |
| unregistered format | tag name is neither built-in nor registered (see below) |

## Tag syntax

| Tag | Behavior |
| --- | --- |
| `mask:"phone"` | format name: built-in or registered via `RegisterMask` |
| `mask:"3,4"` | generic keep: keep first 3 and last 4 runes, mask the middle |
| `mask:"*"` | mask everything (same length) — alias of `password` |
| `mask:"-"` | skip entirely: not masked, and nested fields are **not** recursed |
| none | not masked; nested structs (or struct pointers / struct slices) are still recursed |

- Generic parameters must be two non-negative integers separated by a comma (whitespace allowed, e.g. `"3, 4"`). Invalid values become a build-time config error reported on the first `Mask` call.
- Non-`string` fields with a `mask` tag are skipped silently (no masking, no error); nested struct types are still recursed.
- Unexported fields are always skipped.
- An unregistered format name yields `字段 X: 格式 "y" 未注册` on the first call, for every call until the format is registered (multiple errors joined with `; `). **Other fields are masked normally.**

## Built-in formats

| Format | Rule | Example |
| --- | --- | --- |
| `phone` | keep first 3 + last 4 | `13800138000` → `138****8000` |
| `idcard` | keep first 6 + last 4 | `110101199003071234` → `110101********1234` |
| `bankcard` | keep first 4 + last 4 | `6222020200112233445` → `6222***********3445` |
| `email` | keep first char of local part, keep domain | `zhangsan@example.com` → `z*******@example.com` |
| `name` | keep first char; single-char names unchanged | `张三` → `张*`; `张三四` → `张**`; `王` → `王` |
| `address` | keep first 6 chars | `北京市朝阳区xx路` → `北京市朝阳区***` |
| `password` | mask everything, keep length | `secret123` → `*********` |
| `*` | same as `password` | `abc123` → `******` |

Common rules: everything is **rune-based**; strings shorter than the sum of kept head+tail are fully masked (`phone` on `"123"` → `***`); `email` without `@` is fully masked, a 1-char local part becomes `*@domain`.

## Nesting & batches

Struct fields, struct pointer fields and struct slice fields are recursed automatically — no tag needed on the container:

```go
type Card struct {
	No  string `mask:"bankcard"` // 6222020200112233445 → 6222***********3445
	Cvv string `mask:"*"`        // 123 → ***
}

type User struct {
	Name string `mask:"name"`
	Card Card   // no tag: recursed
}

type Order struct {
	User    *User  // struct pointer: recursed in place
	Cards   []Card // struct slice: recursed element by element
	Comment string
}

order := Order{
	User:  &User{Name: "张三", Card: Card{No: "6222020200112233445", Cvv: "123"}},
	Cards: []Card{{No: "6222020200112233", Cvv: "456"}},
}
mask.Mask(&order)
// order.User.Name    == "张*"
// order.User.Card.No == "6222***********3445"
// order.Cards[0].No  == "6222********2233"
```

Top-level batches — `Mask` accepts a slice pointer (array pointers work too) and processes the whole batch:

```go
users := []User{{Name: "张三"}, {Name: "李四"}}
mask.Mask(&users)
// users[0].Name == "张*", users[1].Name == "李*"
```

nil elements and non-struct elements are skipped.

## Generic keep format

`mask:"n,m"` keeps the first `n` and last `m` runes and masks the middle; `n` or `m` of 0 keeps only one end.

| Tag | Input | Output |
| --- | --- | --- |
| `mask:"4,4"` | `6222020200112233` | `6222********2233` |
| `mask:"0,4"` | `abcdefgh` | `****efgh` |
| `mask:"6,0"` | `北京市朝阳区xx路` | `北京市朝阳区***` |
| `mask:"3,4"` | `123` (too short) | `***` |

## Custom formats

```go
// License plates: keep the first 2 and last 2 runes (Chinese province char safe)
mask.RegisterMask("carplate", func(s string) string {
	r := []rune(s)
	if len(r) < 4 {
		return s
	}
	return string(r[:2]) + "***" + string(r[len(r)-2:])
})

type Car struct {
	Plate string `mask:"carplate"` // 京A12345 → 京A***45
}
```

**Registration semantics:**

- `Formatter` is `func(s string) string` — the whole policy is yours.
- The name is what goes in the tag; an empty name or nil fn is ignored.
- **Takes effect immediately**: the format registry is copy-on-write (`atomic.Pointer`), and registering invalidates the per-type config cache — already-masked types, and even types that previously failed with "format not registered", pick up the new format on their next call.
- Registration is global; do it at startup. It is safe to call concurrently with `Mask` (lock-free reads).

## Working with the family

`mask` shares tag conventions with the author's `validator`, `dict_trans` and `copier` libraries. The tags never collide (`validate:` / `mask:` / `dict:` / `copier:`), so they coexist on the same struct. Recommended pipeline: **validate → business → translate → mask at the boundary → copy to DTO**.

```go
package main

import (
	"fmt"

	"github.com/MouXiaoJun/copier"
	dict "github.com/MouXiaoJun/dict_trans" // module dict_trans declares package dict
	"github.com/MouXiaoJun/mask"
	"github.com/MouXiaoJun/validator"
)

type User struct {
	Name    string `validate:"required" mask:"name"`
	Sex     string `dict:"sex" dictField:"SexName"`
	SexName string
	Phone   string `validate:"len=11" mask:"phone"`
}

func main() {
	dict.RegisterDict("sex", map[string]string{"1": "男", "2": "女"})

	user := User{Name: "张三", Sex: "1", Phone: "13800138000"}

	// 1. validate first
	if err := validator.Validate(&user); err != nil {
		panic(err)
	}
	// 2. translate dict values
	if err := dict.Translate(&user); err != nil {
		panic(err)
	}
	// 3. mask at the boundary
	if err := mask.Mask(&user); err != nil {
		panic(err)
	}
	// 4. copy to a DTO
	var dto User
	if err := copier.Copy(&dto, &user); err != nil {
		panic(err)
	}
	fmt.Println(dto.Name, dto.SexName, dto.Phone) // 张* 男 138****8000
}
```

## Edge cases & limitations

- **In-place semantics**: `Mask` mutates the struct you pass — it returns no copy. Copy first if you need the original (a value copy or via `copier`).
- **Only `string` fields are masked**: non-string fields with a tag are skipped silently — no masking, no error, no false positives.
- **`mask:"-"` skips everything**, including nested recursion on that field.
- Unexported fields are skipped.
- Nil pointers (fields and slice elements) are skipped; never panics.
- Top-level argument must be a struct pointer or a struct slice pointer, else `ErrNotPointer` / `ErrNotStruct`.
- An unregistered format is an error, not a warning: it surfaces on the first call and keeps failing until registered; other fields are still masked.
- Registering a format invalidates the type cache — previously masked types rebuild their config on the next call (by design).
- Concurrency: `Mask` / `MaskOf` are safe to call concurrently; `RegisterMask` is copy-on-write and safe alongside masking, but prefer startup-time registration.
- Idempotent: already-masked `*` stays `*`, so calling `Mask` again changes nothing; mask once at the boundary in production.

## Performance

Apple M5, `go test -bench . -benchmem` (numbers fluctuate ±20%):

| Benchmark | Time | Allocs |
| --- | --- | --- |
| Simple struct (6 string fields masked) | 452 ns/op | 9 allocs/op |
| Nested struct | 565 ns/op | 12 allocs/op |
| 100-element slice | 42 µs/op | 1000 allocs/op |

How: the field config (only fields with a `mask` tag or requiring recursion) is cached per `reflect.Type`; after the first call there is zero tag parsing and zero format-name lookup. The cache is an `atomic.Pointer` with copy-on-write — lock-free reads, invalidated once on `RegisterMask`.

Reproduce locally:

```bash
go test -bench . -benchmem
```

## FAQ

**1. Why not just hand-write the masking?**

Hand-rolled `strings.Repeat` + slicing duplicates logic at every call site, breaks on multi-byte UTF-8, and a rule change means a global search-and-replace. With `mask`, rules live in tags, formats live in one registry, nesting and slices are recursed automatically, `Mask(&users)` handles a batch in one call, and a misspelled format name fails loudly on first use. A simple 6-field struct costs 452 ns / 9 allocs.

**2. Why only `string` fields?**

Masking means "hide text-shaped sensitive data" — phone numbers, IDs, names, addresses, tokens — all strings. What "masking" means for a number or a time is business-specific, so the library never guesses: tagged non-string fields are skipped silently. It also keeps `Formatter` as the minimal `func(s string) string`.

**3. What does "rune-safe" mean?**

Go strings are UTF-8 byte sequences; a Chinese character is 3 bytes, so byte slicing (`s[:2]`) produces mojibake. The library converts to `[]rune` (Unicode code points) before counting, slicing and padding with `*` — `张三` keeps the whole character 「张」 and emoji survive intact. That is why the custom-format example converts to runes too.

**4. How do masking, translation and validation fit together?**

The family tags never collide: `validate:"required,len=11" mask:"phone"` validates the input and masks the output of the same field; `dict:"sex" dictField:"SexName"` translates values. Recommended order: validate (in) → business → translate → mask (out). Full example above.

**5. Does calling `Mask` twice double-mask?**

No. Every built-in format is idempotent on already-masked output: `*` stays `*`, kept head/tail chars are kept again (`138****8000` → `138****8000`). Still, mask once at the boundary for clarity.

## Changelog

See [CHANGELOG.md](./CHANGELOG.md).

## License

[MulanPSL-2.0](LICENSE)
