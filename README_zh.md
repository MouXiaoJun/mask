# go-mask

[English](README.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/MouXiaoJun/mask.svg)](https://pkg.go.dev/github.com/MouXiaoJun/mask)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg?style=flat-square)](LICENSE)
[![GitHub release](https://img.shields.io/github/release/MouXiaoJun/mask.svg?style=flat-square)](https://github.com/MouXiaoJun/mask/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/MouXiaoJun/mask?style=flat-square)](https://goreportcard.com/report/github.com/MouXiaoJun/mask)

一个**零依赖**的 Go 结构体数据脱敏库：struct tag 驱动、就地修改、内置高频格式、嵌套递归，一行 tag 即可在出口处把敏感字段掩掉。

> 🎯 **一行声明式脱敏**：`mask:"phone"` 或 `mask:"3,4"`，格式名即 tag，无需手写循环与下标
>
> ⚡ **就地修改**：`Mask(&user)` 直接改字段值，无需重建对象、无需返回值
>
> 🌏 **中文 / emoji 安全**：全部按 rune 处理，姓名、地址、车牌不会出现半个字符的乱码
>
> 🧩 **家族库**：与同作者的 `validator`（校验）、`dict_trans`（翻译）、`copier`（复制）tag 互不冲突，可先校验后脱敏

## 特性

- ✅ **零依赖**：仅使用 Go 标准库（`reflect` / `sync` / `sync/atomic`），无任何第三方依赖
- ✅ **tag 驱动**：`mask:"phone"` 一行脱敏，格式名即 tag，声明式、可集中审计
- ✅ **就地修改**：`Mask(&user)` 直接修改传入结构体的字段值，不重建对象
- ✅ **内置 8 种格式**：手机号 / 身份证 / 银行卡 / 邮箱 / 姓名 / 地址 / 密码 / 通配
- ✅ **通用保留格式**：`mask:"3,4"` 保留前 3 后 4 个字符，任意 n、m 组合
- ✅ **嵌套递归**：结构体、结构体指针、结构体切片自动递归，无需逐层处理
- ✅ **顶层批量**：`Mask(&users)` 一次脱敏整批切片数据
- ✅ **自定义格式**：`RegisterMask("carplate", fn)` 一行注册，注册后即时生效（含已脱敏过的类型）
- ✅ **中文安全**：按 rune（Unicode 码点）处理，姓名 / 地址 / 车牌等中文不截断
- ✅ **按类型缓存配置**：首次脱敏后零 tag 解析、零格式名查找，读路径无锁
- ✅ **泛型入口**：`MaskOf(&user)` 编译期保证 `*T`
- ✅ **错误可见**：格式名拼错在首次调用即报错，而不是静默漏脱敏；`ErrNotPointer` / `ErrNotStruct` 语义明确

## 安装

需要 Go 1.21+：

```bash
go get github.com/MouXiaoJun/mask
```

## 快速开始

### 完整示例

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
	Note     string // 无 tag：不脱敏
}

func main() {
	user := User{
		Name:     "张三",
		Phone:    "13800138000",
		IDCard:   "110101199003071234",
		Email:    "zhangsan@example.com",
		Password: "secret123",
		Note:     "这条不会被动",
	}

	if err := mask.Mask(&user); err != nil {
		panic(err)
	}

	fmt.Println(user.Name)     // 张*
	fmt.Println(user.Phone)    // 138****8000
	fmt.Println(user.IDCard)   // 110101********1234
	fmt.Println(user.Email)    // z*******@example.com
	fmt.Println(user.Password) // *********
	fmt.Println(user.Note)     // 这条不会被动
}
```

### 泛型入口 `MaskOf`

`Mask` 接受 `any`，误传值类型时会在运行时才报 `ErrNotPointer`；`MaskOf` 把指针要求提前到编译期：

```go
if err := mask.MaskOf(&user); err != nil {
	panic(err)
}
// mask.MaskOf(user)  // ✗ 编译不过：T 是 User，需要 *User
```

### 错误处理

`Mask` / `MaskOf` 的参数错误可用 `errors.Is` 判断；配置错误则提供字段路径和格式名：

| 错误 | 触发场景 |
| --- | --- |
| `mask.ErrNotPointer` | 传入的不是非 nil 指针（值类型、`nil`、`(*T)(nil)`） |
| `mask.ErrNotStruct` | 指针指向的不是结构体或结构体切片（如 `*int`） |
| 未注册格式错误 | tag 里的格式名既非内置也非注册（详见「Tag 语法」） |

```go
var user User
if err := mask.Mask(user); errors.Is(err, mask.ErrNotPointer) {
	// 忘了取地址
}

var n int
if err := mask.Mask(&n); errors.Is(err, mask.ErrNotStruct) {
	// 只想脱敏结构体
}
```

## Tag 语法详解

在结构体字段上写 `mask` tag 即可声明脱敏规则：

| 写法 | 作用 |
| --- | --- |
| `mask:"phone"` | 格式名：使用内置格式或 `RegisterMask` 注册的格式 |
| `mask:"3,4"` | 通用保留：保留前 3 个与后 4 个字符（按 rune），中间用 `*` 掩掉 |
| `mask:"*"` | 全掩：整段替换为等长的 `*`（等价于 `password`） |
| `mask:"-"` | 忽略：不脱敏，**也不递归**其内部嵌套 |
| 无 tag | 不脱敏；若字段本身是嵌套结构体（或结构体指针 / 结构体切片），仍自动递归 |

补充细则：

- **通用参数必须是两个非负整数**，逗号分隔，允许空格（`"3, 4"` 合法）；负数、非数字、多于一个逗号 → 构建期记录配置错误，首次调用 `Mask` 时报错。
- **非 `string` 字段带 tag**：值不脱敏，但 tag 仍会校验；若该字段的类型是嵌套结构体，仍递归其内部（除非使用 `mask:"-"`）。
- **未导出字段**：一律跳过（不脱敏、不递归）。
- **未注册的格式名**：构建期记录配置错误，首次调用返回 `字段 X: 格式 "y" 未注册`；同一类型每次调用都会返回该错误，直到用 `RegisterMask` 注册后才自动修复（见「自定义格式」）。**其他字段不受影响，照常脱敏。**

## 内置格式总表

内置 8 种格式，覆盖国内业务最常见的敏感字段：

| 格式 | 规则 | 示例 |
| --- | --- | --- |
| `phone` | 保留前 3 后 4 | `13800138000` → `138****8000` |
| `idcard` | 保留前 6 后 4 | `110101199003071234` → `110101********1234` |
| `bankcard` | 保留前 4 后 4 | `6222020200112233445` → `6222***********3445` |
| `email` | 本地部分保留首字符，域名保留 | `zhangsan@example.com` → `z*******@example.com` |
| `name` | 保留首字符，其余掩掉；单字不掩 | `张三` → `张*`；`张三四` → `张**`；`王` → `王` |
| `address` | 保留前 6 个字符 | `北京市朝阳区xx路` → `北京市朝阳区***` |
| `password` | 全掩，保留原长度 | `secret123` → `*********` |
| `*` | 同 `password` | `abc123` → `******` |

三条通用规则：

- **全部格式按 rune 处理**：`maskName("李小明")` 保留的是整个「李」字，绝不会切出半个字符。
- **字符串过短（不足首尾之和）时全掩**：如 `phone` 遇到 `"123"` → `***`，不会输出畸形的残缺号码。
- **特殊输入**：`name` 对单字（`"王"`）不掩；`email` 不含 `@` 视为非法邮箱直接全掩，本地部分只有 1 个字符时输出 `*@domain`。

## 嵌套与切片

### 嵌套递归

结构体字段、结构体指针字段、结构体切片字段（含数组）自动递归脱敏，无 tag 也能透传：

```go
type Card struct {
	No  string `mask:"bankcard"` // 6222020200112233445 → 6222***********3445
	Cvv string `mask:"*"`        // 123 → ***
}

type User struct {
	Name string `mask:"name"`
	Card Card   // 无 tag 的嵌套结构体：自动递归
}

type Order struct {
	User    *User  // 结构体指针：递归到指针指向的对象（原地修改）
	Cards   []Card // 结构体切片：逐元素递归
	Comment string // 普通字段：不脱敏
}

order := Order{
	User:    &User{Name: "张三", Card: Card{No: "6222020200112233445", Cvv: "123"}},
	Cards:   []Card{{No: "6222020200112233", Cvv: "456"}},
	Comment: "保留",
}
mask.Mask(&order)
// order.User.Name      == "张*"
// order.User.Card.No   == "6222***********3445"
// order.Cards[0].No    == "6222********2233"
// order.Comment        == "保留"
```

### 顶层批量脱敏

`Mask` 接受结构体**切片指针**（数组指针亦可），一次调用处理整批数据，适合分页列表、导出等场景：

```go
users := []User{
	{Name: "张三"},
	{Name: "李四"},
}
mask.Mask(&users)
// users[0].Name == "张*"
// users[1].Name == "李*"
```

元素可以是结构体或结构体指针；nil 元素、最终不是结构体的元素静默跳过。

## 通用保留格式

`mask:"n,m"` 是内置格式之外的万能兜底：保留前 `n` 个、后 `m` 个字符（按 rune），中间用 `*` 掩掉。`n` 或 `m` 为 0 时退化为只保留一端。

| tag | 输入 | 输出 |
| --- | --- | --- |
| `mask:"4,4"` | `6222020200112233` | `6222********2233` |
| `mask:"0,4"` | `abcdefgh` | `****efgh` |
| `mask:"6,0"` | `北京市朝阳区xx路` | `北京市朝阳区***` |
| `mask:"3,4"` | `123`（过短） | `***` |

```go
type Account struct {
	No       string `mask:"4,4"` // 6222020200112233 → 6222********2233
	Serial   string `mask:"0,4"` // abcdefgh → ****efgh
	Region   string `mask:"6,0"` // 北京市朝阳区xx路 → 北京市朝阳区***
	TooShort string `mask:"3,4"` // 123 → ***（不足 3+4，全掩）
}

a := Account{No: "6222020200112233", Serial: "abcdefgh", Region: "北京市朝阳区xx路", TooShort: "123"}
mask.Mask(&a)
// a.No       == "6222********2233"
// a.Serial   == "****efgh"
// a.Region   == "北京市朝阳区***"
// a.TooShort == "***"
```

参数非法（负数、非数字、多逗号）会被记录为配置错误，首次调用时报错，而不是静默忽略。

## 自定义格式

内置格式不够用时，`RegisterMask` 一行注册自定义格式：

```go
// 车牌脱敏：保留前 2 后 2 个字符（按 rune，中文省份字安全）
mask.RegisterMask("carplate", func(s string) string {
	r := []rune(s)
	if len(r) < 4 {
		return s // 太短不掩
	}
	return string(r[:2]) + "***" + string(r[len(r)-2:])
})

type Car struct {
	Plate string `mask:"carplate"` // 京A12345 → 京A***45
}

car := Car{Plate: "京A12345"}
mask.Mask(&car) // car.Plate == "京A***45"
```

**注册语义（重要）：**

- `Formatter` 签名是 `func(s string) string`：输入原文、输出脱敏结果，怎么处理完全由你决定。
- 格式名即 tag 里的名字；`name` 为空或 `fn` 为 nil 时忽略。
- **注册后即时生效**：注册表是写时复制（`atomic.Pointer`），注册会同时使按类型缓存的配置失效——之前已经脱敏过的类型、甚至之前因「格式未注册」报错的类型，下一次调用都会自动用上新格式。
- 注册是**全局**的，建议在程序启动期完成；与 `Mask` 并发调用也是安全的（读路径无锁）。

## 与家族库配合：先校验后脱敏

`mask` 与同作者的 `validator`（校验）、`dict_trans`（字典翻译）、`copier`（结构体复制）同属一套 tag 驱动的零依赖家族。各库 tag 名互不冲突（`validate:` / `mask:` / `dict:` / `copier:`），可以在同一个结构体上共存。

推荐的处理顺序：**校验 → 业务处理 → 翻译 → 出口前脱敏 → 拷贝 DTO**。脱敏放在出口前最后一步，保证日志、链路、内部逻辑全程可见原文，只有对外输出被掩掉。

```go
package main

import (
	"fmt"

	"github.com/MouXiaoJun/copier"
	dict "github.com/MouXiaoJun/dict_trans" // 该模块的包名即 dict
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
	// 注册字典（dict_trans）
	dict.RegisterDict("sex", map[string]string{"1": "男", "2": "女"})

	user := User{Name: "张三", Sex: "1", Phone: "13800138000"}

	// 1. 先校验（validator）：入参不合法直接返回，不进入脱敏
	if err := validator.Validate(&user); err != nil {
		panic(err)
	}
	// 2. 字典翻译（dict_trans）
	if err := dict.Translate(&user); err != nil {
		panic(err)
	}
	// 3. 出口前脱敏（mask）
	if err := mask.Mask(&user); err != nil {
		panic(err)
	}
	// 4. 拷贝到 DTO（copier）
	var dto User
	if err := copier.Copy(&dto, &user); err != nil {
		panic(err)
	}
	fmt.Println(dto.Name, dto.SexName, dto.Phone) // 张* 男 138****8000
}
```

需要 `go get` 家族其余成员：`github.com/MouXiaoJun/validator`、`github.com/MouXiaoJun/dict_trans`、`github.com/MouXiaoJun/copier`。

## 边界与限制

- **就地修改**：`Mask` 直接修改传入结构体的字段值，**不返回副本**。需要保留原数据时，先拷贝一份再脱敏（值拷贝或配合 `copier` 复制到 DTO）。
- **只处理 `string` 字段**：数字、时间等非字符串值不脱敏，但 tag 仍会校验；无效 tag 会报告配置错误。
- **`mask:"-"` 是完全忽略**：不脱敏，同时关闭该字段的嵌套递归——不要用 `-` 去「放行嵌套」。
- **未导出字段跳过**：小写字段不脱敏也不递归。
- **nil 指针跳过**：nil 的指针字段、切片中的 nil 元素都不处理，不 panic。
- **顶层参数校验**：必须是结构体指针或结构体切片指针，否则返回 `ErrNotPointer` / `ErrNotStruct`；顶层切片元素最终不是结构体时静默跳过。
- **未注册格式是错误而非警告**：格式名拼错会在首次调用时暴露（`字段 X: 格式 "y" 未注册`），X 为完整 Go 字段/索引路径，例如 `User.Cards[1].No`、顶层批次的 `[0].Phone`；共享对象只报告首次访问路径。多个错误用 `; ` 拼接；注册后下次调用自动修复。其他字段仍会脱敏，返回错误不会回滚已完成的修改。
- **注册格式使缓存失效**：`RegisterMask` 之后，之前脱敏过的类型下次调用会重建配置——这是特性，不是 bug。
- **并发安全**：`Mask` / `MaskOf` 可并发处理不同对象，不可并发修改同一个对象；`RegisterMask` 与脱敏可并发（写时复制注册表 + `atomic.Pointer` 缓存），但请把注册放在启动期。
- **幂等**：内置格式保持已掩结果；自定义格式每次调用都会执行，不保证幂等。生产上出口处脱敏一次即可。

## 性能

Apple M5，`go test -bench . -benchmem`（每次波动 ±20%）：

| 基准 | 耗时 | 分配 |
| --- | --- | --- |
| 简单结构体（6 个 string 字段脱敏） | 452 ns/op | 9 allocs/op |
| 嵌套结构体 | 565 ns/op | 12 allocs/op |
| 100 元素切片 | 42 µs/op | 1000 allocs/op |

怎么做到的：**按类型缓存字段配置**（只收录有 `mask` tag 或需递归的字段，缓存键是 `reflect.Type`），首次脱敏后零 tag 解析、零格式名查找；缓存读写用 `atomic.Pointer` 写时复制，读路径无锁，`RegisterMask` 时整表失效一次。格式函数内部也尽量少分配（`phone` / `name` / `password` 等按 rune 后一次拼接）。

自己复现：

```bash
go test -bench . -benchmem
```

## FAQ

**1. 和手写脱敏比有什么优势？**

手写 `strings.Repeat` + 下标切片的方案每处都复制一份逻辑，格式一改就要全局搜索替换，还容易在中文上按字节切出乱码。用 `mask`：规则集中在 tag 上、格式集中在一个注册表里，一处注册全局生效；嵌套结构体、指针、切片自动递归，`Mask(&users)` 一次处理整批；按类型缓存后单结构体 452 ns / 9 allocs，比手写反射实现快得多；格式名拼错首次调用就报错，不会静默漏脱敏。

**2. 为什么只处理 string 字段？**

脱敏的语义就是「把文本类敏感信息掩掉」：手机号、身份证、姓名、地址、token……全部是字符串。数字、时间等结构化字段的「脱敏」语义（保留几位？取模？打散？）取决于业务，库不做越权的猜测——带 tag 的非 string 字段静默跳过，绝不误伤。这也让 `Formatter` 保持最简签名 `func(s string) string`。

**3. rune 安全是怎么回事？**

Go 字符串是 UTF-8 字节序列，一个汉字占 3 个字节。按字节切片（`s[:2]`）会把中文拦腰截断成乱码。库内部全部先转 `[]rune`（Unicode 码点）再计数、切片、拼 `*`，所以 `张三` 保留的是完整的「张」，emoji 也不会被切开。这也是自定义格式示例里车牌（`京A12345`）必须转 rune 的原因。

**4. 脱敏和翻译 / 校验怎么一起用？**

家族库 tag 名互不冲突，可以挂在同一字段上：`validate:"required,len=11" mask:"phone"` 同时完成「入参校验」和「出口脱敏」，`dict:"sex" dictField:"SexName"` 负责翻译。推荐顺序：校验（进）→ 业务 → 翻译 → 脱敏（出）。完整示例见「与家族库配合」。

**5. 重复调用 `Mask` 会重复脱敏吗？**

不会。所有内置格式对已掩结果都是幂等的：`*` 会保持为 `*`，保留的前后字符再次保留。比如 `138****8000` 再脱敏一次还是 `138****8000`。不过生产上建议在出口处只脱敏一次，语义更清晰。

## CHANGELOG

维护范围保持在现有 struct tag 脱敏、错误诊断和 Go 兼容性，不扩成数据库匿名化平台。
CI 覆盖 Go 1.21.0、1.25.x 与 stable 的构建、vet、全量测试、race、覆盖率及短时 fuzz。

版本历史与变更见 [CHANGELOG.md](./CHANGELOG.md)。

## License

[MIT](LICENSE)
