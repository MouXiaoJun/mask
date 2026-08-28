# go-mask

[English](README.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/MouXiaoJun/mask.svg)](https://pkg.go.dev/github.com/MouXiaoJun/mask)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MulanPSL--2.0-green.svg?style=flat-square)](LICENSE)

一个**零依赖**的 Go 结构体数据脱敏库：struct tag 驱动、就地修改、内置高频格式、嵌套递归。

## 特性

- ✅ **零依赖**：仅使用 Go 标准库
- ✅ **tag 驱动**：`mask:"phone"` 一行脱敏，格式名即 tag
- ✅ **就地修改**：`Mask(&user)` 直接脱敏字段值，无需重建对象
- ✅ **内置格式**：手机号 / 身份证 / 银行卡 / 邮箱 / 姓名 / 地址 / 密码 / 通配
- ✅ **通用保留**：`mask:"3,4"` 保留前 3 后 4 个字符
- ✅ **嵌套递归**：结构体、结构体指针、结构体切片自动递归
- ✅ **顶层切片**：`Mask(&users)` 一次脱敏整批数据
- ✅ **自定义格式**：`RegisterMask` 一行注册
- ✅ **中文安全**：按 rune 处理，姓名/地址等中文不截断

## 快速开始

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
}

func main() {
	user := User{
		Name: "张三", Phone: "13800138000",
		IDCard: "110101199003071234",
		Email:  "zhangsan@example.com", Password: "secret",
	}
	mask.Mask(&user)
	fmt.Println(user.Phone) // 138****8000
	fmt.Println(user.Name)  // 张*
}
```

## Tag 语法

| tag | 行为 |
| --- | --- |
| `mask:"phone"` | 使用内置/注册格式 |
| `mask:"3,4"` | 通用格式：保留前 3 与后 4 个字符，中间掩掉 |
| `mask:"*"` | 全部掩掉（保留原长度） |
| `mask:"-"` | 忽略该字段 |
| 无 tag | 不脱敏；若是嵌套结构体仍递归 |

非字符串字段带 `mask` tag 时静默跳过（不脱敏、不报错）。

## 内置格式

| 格式 | 规则 | 示例 |
| --- | --- | --- |
| `phone` | 保留前 3 后 4 | `13800138000` → `138****8000` |
| `idcard` | 保留前 6 后 4 | `110101199003071234` → `110101********1234` |
| `bankcard` | 保留前 4 后 4 | `6222020200112233` → `6222********2233` |
| `email` | 本地部分保留首字符 | `zhangsan@example.com` → `z*******@example.com` |
| `name` | 保留首字符，其余掩 | `张三` → `张*`，`张三四` → `张**` |
| `address` | 保留前 6 个字符 | `北京市朝阳区xx路` → `北京市朝阳区**` |
| `password` | 全部掩掉 | `secret123` → `*********` |
| `*` | 同 password | 同上 |

## 嵌套与切片

```go
type Order struct {
	Buyer User   // 嵌套结构体自动递归
	Cards []Card // 结构体切片逐个脱敏
}

// 顶层切片：一次脱敏整批
users := []User{{Name: "张三"}, {Name: "李四"}}
mask.Mask(&users) // users[0].Name = "张*"
```

## 自定义格式

```go
mask.RegisterMask("carplate", func(s string) string {
	r := []rune(s) // 中文安全：用 rune 而非字节切片
	if len(r) < 4 {
		return s
	}
	return string(r[:2]) + "***" + string(r[len(r)-2:])
})

type Car struct{ Plate string `mask:"carplate"` }
```

注册后已脱敏过的类型也会自动用上新格式（类型配置缓存写时失效）。

## 性能

Apple M5，`go test -bench . -benchmem`：

| 基准 | 耗时 | 分配 |
| --- | --- | --- |
| 简单结构体（6 字段脱敏） | 452 ns/op | 9 次 |
| 嵌套结构体 | 565 ns/op | 12 次 |
| 100 元素切片 | 42 µs/op | 1000 次 |

按类型缓存字段配置，首次脱敏后零 tag 解析与格式名查找。

## 安装

```bash
go get github.com/MouXiaoJun/mask
```

## License

MulanPSL-2.0
