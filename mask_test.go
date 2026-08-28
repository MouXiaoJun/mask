package mask

import (
	"strings"
	"testing"
)

func TestMaskPhone(t *testing.T) {
	if got := maskPhone("13800138000"); got != "138****8000" {
		t.Fatalf("phone = %q", got)
	}
	// 太短：全掩
	if got := maskPhone("123"); got != "***" {
		t.Fatalf("short phone = %q", got)
	}
}

func TestMaskIDCard(t *testing.T) {
	if got := maskIDCard("110101199003071234"); got != "110101********1234" {
		t.Fatalf("idcard = %q", got)
	}
}

func TestMaskBank(t *testing.T) {
	// 19 字符，保留前 4 后 4，中间 11 个 *
	if got := maskBank("6222020200112233445"); got != "6222***********3445" {
		t.Fatalf("bank = %q", got)
	}
}

func TestMaskEmail(t *testing.T) {
	// zhangsan 8 字符：保留首字符，其余 7 个 *
	if got := maskEmail("zhangsan@example.com"); got != "z*******@example.com" {
		t.Fatalf("email = %q", got)
	}
	if got := maskEmail("a@b.com"); got != "*@b.com" {
		t.Fatalf("short email = %q", got)
	}
	if got := maskEmail("no-at-sign"); got != "**********" { // 10 字符
		t.Fatalf("invalid email = %q", got)
	}
}

func TestMaskName(t *testing.T) {
	if got := maskName("张三"); got != "张*" {
		t.Fatalf("2-char name = %q", got)
	}
	if got := maskName("张三四"); got != "张**" {
		t.Fatalf("3-char name = %q", got)
	}
	if got := maskName("王"); got != "王" {
		t.Fatalf("1-char name = %q", got)
	}
}

func TestMaskAll(t *testing.T) {
	if got := maskAll("secret123"); got != "*********" {
		t.Fatalf("maskAll = %q", got)
	}
}

func TestKeepBoth(t *testing.T) {
	if got := keepBoth(3, 4)("13800138000"); got != "138****8000" {
		t.Fatalf("keepBoth = %q", got)
	}
	if got := keepBoth(0, 4)("abcdefgh"); got != "****efgh" {
		t.Fatalf("keepBoth tail-only = %q", got)
	}
	// 不足首尾之和：全掩
	if got := keepBoth(3, 4)("abc"); got != "***" {
		t.Fatalf("keepBoth short = %q", got)
	}
}

func TestMaskStruct(t *testing.T) {
	type User struct {
		Name     string `mask:"name"`
		Phone    string `mask:"phone"`
		IDCard   string `mask:"idcard"`
		Email    string `mask:"email"`
		Password string `mask:"password"`
		Note     string // 无 tag 不脱敏
	}
	u := User{Name: "张三", Phone: "13800138000", IDCard: "110101199003071234",
		Email: "zhangsan@example.com", Password: "secret", Note: "保留"}
	if err := Mask(&u); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if u.Name != "张*" || u.Phone != "138****8000" || u.IDCard != "110101********1234" ||
		u.Email != "z*******@example.com" || u.Password != "******" || u.Note != "保留" {
		t.Fatalf("mask result wrong: %+v", u)
	}
}

func TestMaskGenericKeep(t *testing.T) {
	type Account struct {
		No string `mask:"4,4"` // 通用：保留前 4 后 4
	}
	a := Account{No: "6222020200112233"}
	if err := Mask(&a); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if a.No != "6222********2233" {
		t.Fatalf("generic keep wrong: %q", a.No)
	}
}

func TestMaskIgnore(t *testing.T) {
	type Req struct {
		Skip string `mask:"-"`
		OK   string `mask:"phone"`
	}
	r := Req{Skip: "保留", OK: "13800138000"}
	if err := Mask(&r); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if r.Skip != "保留" || r.OK != "138****8000" {
		t.Fatalf("ignore wrong: %+v", r)
	}
}

func TestMaskWildcard(t *testing.T) {
	type Req struct {
		Token string `mask:"*"`
	}
	r := Req{Token: "abc123"}
	if err := Mask(&r); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if r.Token != "******" {
		t.Fatalf("wildcard wrong: %q", r.Token)
	}
}

func TestMaskNested(t *testing.T) {
	type Card struct {
		No string `mask:"bankcard"`
	}
	type User struct {
		Name string `mask:"name"`
		Card Card   // 嵌套
	}
	u := User{Name: "张三", Card: Card{No: "6222020200112233445"}}
	if err := Mask(&u); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if u.Card.No != "6222***********3445" {
		t.Fatalf("nested mask wrong: %q", u.Card.No)
	}
}

func TestMaskSliceAndPtr(t *testing.T) {
	type User struct {
		Name string `mask:"name"`
	}
	u1, u2 := User{Name: "张三"}, User{Name: "李四"}
	slice := []User{u1, u2}
	if err := Mask(&slice); err != nil {
		t.Fatalf("Mask slice failed: %v", err)
	}
	if slice[0].Name != "张*" || slice[1].Name != "李*" {
		t.Fatalf("slice mask wrong: %+v", slice)
	}

	ptr := &User{Name: "王五"}
	type Holder struct {
		User *User
	}
	h := Holder{User: ptr}
	if err := Mask(&h); err != nil {
		t.Fatalf("Mask ptr failed: %v", err)
	}
	if h.User.Name != "王*" {
		t.Fatalf("ptr mask wrong: %+v", h)
	}
}

func TestMaskErrors(t *testing.T) {
	type Req struct{ Name string }
	var req Req
	if err := Mask(req); err == nil || !strings.Contains(err.Error(), "must be a non-nil pointer") {
		t.Fatalf("non-ptr err = %v", err)
	}
	if err := Mask(nil); err == nil {
		t.Fatal("nil should error")
	}
	var p *Req
	if err := Mask(p); err == nil {
		t.Fatal("nil ptr should error")
	}
	var n int
	if err := Mask(&n); err == nil {
		t.Fatal("non-struct should error")
	}
}

func TestMaskUnregisteredFormat(t *testing.T) {
	type Req struct {
		Foo string `mask:"not_a_format"`
	}
	err := Mask(&Req{})
	if err == nil || !strings.Contains(err.Error(), "未注册") {
		t.Fatalf("unregistered format should error: %v", err)
	}
}

func TestMaskNonStringField(t *testing.T) {
	// 非 string 字段带 mask tag：不脱敏也不报错
	type Req struct {
		Age int `mask:"phone"`
	}
	r := Req{Age: 30}
	if err := Mask(&r); err != nil {
		t.Fatalf("non-string field should be skipped: %v", err)
	}
	if r.Age != 30 {
		t.Fatalf("non-string field mutated: %+v", r)
	}
}

func TestMaskCustomFormat(t *testing.T) {
	type Req struct {
		Plate string `mask:"carplate"`
	}
	RegisterMask("carplate", func(s string) string {
		r := []rune(s)
		if len(r) < 4 {
			return s
		}
		return string(r[:2]) + "***" + string(r[len(r)-2:])
	})
	r := Req{Plate: "京A12345"}
	if err := Mask(&r); err != nil {
		t.Fatalf("Mask failed: %v", err)
	}
	if r.Plate != "京A***45" {
		t.Fatalf("custom format wrong: %q", r.Plate)
	}
}

func TestMaskOfGeneric(t *testing.T) {
	type User struct {
		Phone string `mask:"phone"`
	}
	u := User{Phone: "13800138000"}
	if err := MaskOf(&u); err != nil {
		t.Fatalf("MaskOf failed: %v", err)
	}
	if u.Phone != "138****8000" {
		t.Fatalf("MaskOf wrong: %q", u.Phone)
	}
}
