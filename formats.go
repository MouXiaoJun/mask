package mask

import (
	"strings"
	"sync"
	"sync/atomic"
)

// Formatter 脱敏格式：输入原文，输出脱敏后的字符串。
type Formatter func(s string) string

// formatRegistry 格式注册表：读多写少，写时复制（与 dict_trans 注册表一致）。
// RegisterMask 之后需要使类型配置缓存失效（见 mask.go 的 invalidateCache）。
type formatRegistry struct {
	formats atomic.Pointer[map[string]Formatter]
	mu      sync.Mutex
}

var defaultRegistry = &formatRegistry{}

func (r *formatRegistry) load() map[string]Formatter {
	if m := r.formats.Load(); m != nil {
		return *m
	}
	return nil
}

// register 注册格式（写时复制）。
func (r *formatRegistry) register(name string, fn Formatter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	old := r.load()
	next := make(map[string]Formatter, len(old)+1)
	for k, v := range old {
		next[k] = v
	}
	next[name] = fn
	r.formats.Store(&next)
}

// RegisterMask 注册自定义脱敏格式，name 即 struct tag 里的格式名。
// 注册后已校验过的类型也会用上新格式（类型配置缓存写时失效）。
// 示例：
//
//	RegisterMask("carplate", func(s string) string {
//		if len(s) < 3 { return s }
//		return s[:2] + "***" + s[len(s)-2:]
//	})
//
//	type Car struct{ Plate string `mask:"carplate"` }
func RegisterMask(name string, fn Formatter) {
	if name == "" || fn == nil {
		return
	}
	defaultRegistry.register(name, fn)
	invalidateCache()
}

// formatFor 查格式函数，未注册返回 nil。
func formatFor(name string) Formatter {
	return defaultRegistry.load()[name]
}

// --- 内置格式 ---

// builtinFormats 内置高频脱敏格式，覆盖国内业务最常见的敏感字段。
var builtinFormats = []struct {
	name string
	fn   Formatter
}{
	{"phone", maskPhone},   // 手机号：138****8000
	{"idcard", maskIDCard}, // 身份证：110101********1234
	{"bankcard", maskBank}, // 银行卡：6222********1234
	{"email", maskEmail},   // 邮箱：a***@b.com
	{"name", maskName},     // 姓名：张* / 张**
	{"address", maskAddr},  // 地址：保留前 6 个字符
	{"password", maskAll},  // 密码/密钥：全部掩掉
	{"*", maskAll},         // 通配：全部掩掉
}

func init() {
	for _, b := range builtinFormats {
		defaultRegistry.register(b.name, b.fn)
	}
}

// keepBoth 通用格式：保留前 keepHead 与后 keepTail 个字符（按 rune 计），中间掩掉。
// 字符串太短（不足首尾之和）时全部掩掉。
func keepBoth(head, tail int) Formatter {
	return func(s string) string {
		r := []rune(s)
		if len(r) <= head+tail {
			return strings.Repeat("*", len(r))
		}
		return string(r[:head]) + strings.Repeat("*", len(r)-head-tail) + string(r[len(r)-tail:])
	}
}

func maskPhone(s string) string  { return keepBoth(3, 4)(s) }
func maskIDCard(s string) string { return keepBoth(6, 4)(s) }
func maskBank(s string) string   { return keepBoth(4, 4)(s) }
func maskAddr(s string) string   { return keepBoth(6, 0)(s) }

// maskEmail a***@b.com：本地部分保留首字符，域名保留。
func maskEmail(s string) string {
	at := strings.IndexByte(s, '@')
	if at < 0 {
		return maskAll(s) // 没有 @：不是合法邮箱，全掩
	}
	local, domain := s[:at], s[at:]
	r := []rune(local)
	if len(r) <= 1 {
		return "*" + domain
	}
	return string(r[:1]) + strings.Repeat("*", len(r)-1) + domain
}

// maskName 保留首字符，其余掩掉：张三 → 张*，张三四 → 张**。
func maskName(s string) string {
	r := []rune(s)
	if len(r) <= 1 {
		return s // 单字名不掩
	}
	return string(r[:1]) + strings.Repeat("*", len(r)-1)
}

// maskAll 全部掩掉（保留原长度，密码等场景）。
func maskAll(s string) string {
	return strings.Repeat("*", len([]rune(s)))
}
