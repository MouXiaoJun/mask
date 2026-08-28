package mask

import (
	"strings"
	"testing"
)

// FuzzFormatsNoPanic 任意字符串（含中文/emoji/超长）应用全部内置格式不 panic。
func FuzzFormatsNoPanic(f *testing.F) {
	f.Add("13800138000")
	f.Add("张三")
	f.Add("")
	f.Add(strings.Repeat("a", 10000))
	f.Add("emoji😀测试")
	f.Add("a@b.c")
	f.Fuzz(func(t *testing.T, s string) {
		for _, b := range builtinFormats {
			b.fn(s)
		}
	})
}

// FuzzParseKeep 任意 tag 解析不 panic，返回 true 时参数必须合法。
func FuzzParseKeep(f *testing.F) {
	f.Add("3,4")
	f.Add("0,0")
	f.Add("-1,5")
	f.Add("x,y")
	f.Add("3")
	f.Add("3,4,5")
	f.Fuzz(func(t *testing.T, tag string) {
		h, ta, ok := parseKeep(tag)
		if ok {
			if h < 0 || ta < 0 {
				t.Fatalf("negative keep from %q", tag)
			}
		}
	})
}

// FuzzMaskNoPanic 随机字符串结构体脱敏不 panic。
func FuzzMaskNoPanic(f *testing.F) {
	type Req struct {
		Name  string `mask:"name"`
		Phone string `mask:"phone"`
		Email string `mask:"email"`
		Token string `mask:"*"`
		Keep  string `mask:"2,2"`
	}
	f.Add("张三", "13800138000", "a@b.com", "tok", "abc")
	f.Add("", "", "", "", "")
	f.Add(strings.Repeat("x", 5000), "1", "2", "3", "4")
	f.Fuzz(func(t *testing.T, name, phone, email, token, keep string) {
		r := Req{Name: name, Phone: phone, Email: email, Token: token, Keep: keep}
		_ = Mask(&r)
	})
}
