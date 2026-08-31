package mask

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestMaskUnregisteredFormatPaths(t *testing.T) {
	type Leaf struct {
		Secret string `mask:"missing-path-format"`
	}
	p := &Leaf{}
	pp := &p
	for _, test := range []struct {
		name  string
		value any
		paths []string
	}{
		{"root", &Leaf{}, []string{"Secret"}},
		{"nested", &struct{ Child Leaf }{}, []string{"Child.Secret"}},
		{"embedded", &struct{ Leaf }{}, []string{"Leaf.Secret"}},
		{"pointer", &struct{ Child ***Leaf }{&pp}, []string{"Child.Secret"}},
		{"slice", &struct{ Items []*Leaf }{[]*Leaf{nil, {}, {}}}, []string{"Items[1].Secret", "Items[2].Secret"}},
		{"array", &struct{ Items [1]Leaf }{}, []string{"Items[0].Secret"}},
		{"root slice", &[]*Leaf{nil, {}}, []string{"[1].Secret"}},
		{"root array", &[1]struct{ Child Leaf }{}, []string{"[0].Child.Secret"}},
		{"shared", &struct{ First, Second *Leaf }{p, p}, []string{"First.Secret"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			messages := make([]string, len(test.paths))
			for i, path := range test.paths {
				messages[i] = fmt.Sprintf("字段 %s: 格式 %q 未注册", path, "missing-path-format")
			}
			want := "mask: " + strings.Join(messages, "; ")
			// Repeat to check cached configurations do not retain a traversal path.
			for i := 0; i < 2; i++ {
				if err := Mask(test.value); err == nil || err.Error() != want {
					t.Fatalf("Mask() = %v, want %q", err, want)
				}
			}
		})
	}

	v := struct {
		Child struct {
			Bad  string `mask:"-1,2"`
			Good string `mask:"*"`
		}
	}{}
	v.Child.Good = "secret"
	if err := MaskOf(&v); err == nil || err.Error() != `mask: 字段 Child.Bad: 格式 "-1,2" 未注册` {
		t.Fatalf("invalid keep format = %v", err)
	}
	if v.Child.Good != "******" {
		t.Fatal("valid fields must still be masked when another field has a configuration error")
	}
}

func TestMaskRetainCountOverflow(t *testing.T) {
	max := strconv.Itoa(int(^uint(0) >> 1))
	for _, format := range []string{max + ",1", "1," + max, max + "," + max, "1,1"} {
		t.Run(format, func(t *testing.T) {
			typ := reflect.StructOf([]reflect.StructField{{
				Name: "Secret", Type: reflect.TypeOf(""), Tag: reflect.StructTag("mask:" + strconv.Quote(format)),
			}})
			v := reflect.New(typ)
			v.Elem().Field(0).SetString("中文")
			if err := Mask(v.Interface()); err != nil {
				t.Fatal(err)
			}
			if got := v.Elem().Field(0).String(); got != "**" {
				t.Fatalf("Secret = %q, want **", got)
			}
		})
	}
}

func TestMaskCyclicAndSharedNodes(t *testing.T) {
	calls := 0
	RegisterMask("cycle-once", func(s string) string {
		calls++
		if calls > 4 {
			panic("the same object was visited repeatedly")
		}
		return s + "!"
	})
	type part struct {
		Secret string `mask:"cycle-once"`
	}
	type node struct {
		Part   part
		Secret string `mask:"cycle-once"`
		Next   *node
		Items  []*node
		Skip   *node `mask:"-"`
	}
	root := &node{Part: part{Secret: "a"}, Secret: "b"}
	child := &node{Part: part{Secret: "c"}, Secret: "d", Next: root}
	root.Next = root
	root.Items = []*node{child, child, nil}
	root.Skip = &node{Secret: "untouched"}
	defer func() {
		if p := recover(); p != nil {
			t.Fatalf("Mask did not terminate safely: %v", p)
		}
	}()
	if err := Mask(root); err != nil {
		t.Fatal(err)
	}
	if calls != 4 || root.Part.Secret != "a!" || root.Secret != "b!" ||
		child.Part.Secret != "c!" || child.Secret != "d!" || root.Skip.Secret != "untouched" {
		t.Fatalf("calls=%d, root=%+v, child=%+v", calls, root, child)
	}
}

func TestMaskDeepPointers(t *testing.T) {
	type child struct {
		Secret string `mask:"*"`
	}
	p := &child{Secret: "secret"}
	pp := &p
	v := struct{ Child ***child }{Child: &pp}
	if err := Mask(&v); err != nil {
		t.Fatal(err)
	}
	if p.Secret != "******" {
		t.Fatalf("Secret = %q", p.Secret)
	}
}

func TestMaskInterfaceFields(t *testing.T) {
	type child struct {
		Secret string `mask:"*"`
	}
	for _, value := range []any{nil, "plain", &child{Secret: "secret"}} {
		v := struct {
			Extra   any
			Skipped any `mask:"-"`
			Items   []any
			Secret  string `mask:"*"`
		}{Extra: value, Skipped: value, Items: []any{value}, Secret: "secret"}
		if err := Mask(&v); err != nil {
			t.Fatal(err)
		}
		if v.Secret != "******" {
			t.Fatalf("Secret = %q", v.Secret)
		}
		if c, ok := value.(*child); ok && c.Secret != "secret" {
			t.Fatal("interface fields should remain outside static traversal")
		}
	}
}
