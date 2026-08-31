// Package mask 提供零依赖的结构体数据脱敏。
//
// struct tag 驱动，就地修改字符串字段：
//   - `mask:"phone"`：使用内置/注册的格式名
//   - `mask:"3,4"`：通用格式，保留前 3 个与后 4 个字符，中间掩掉
//   - `mask:"-"`：忽略该字段
//   - `mask:"*"`：全部掩掉
//
// 嵌套结构体、结构体指针、结构体切片自动递归脱敏。
// 非字符串字段跳过（不脱敏）。
//
// 性能：按类型缓存字段配置（只收录有 mask tag 或需递归的字段），
// 首次脱敏后零 tag 解析与格式名查找。
package mask

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// fieldConfig 单个字段的脱敏配置。
type fieldConfig struct {
	name     string
	format   Formatter // 非 nil 表示该字段需脱敏；nil 表示仅嵌套递归
	isNested bool
}

// typeConfig 按类型缓存的脱敏配置。
type typeConfig struct {
	fields []fieldConfig
	errs   []maskError // 构建期发现的配置错误（如未注册的格式名）
}

type maskError struct {
	field, name string
}

// configCache 按类型缓存：读无锁（atomic Load），写时复制。
var (
	configCache atomic.Pointer[map[reflect.Type]*typeConfig]
	configMu    sync.Mutex
)

// invalidateCache 注册新格式后使类型缓存失效。
func invalidateCache() {
	configCache.Store(nil)
}

func getConfig(t reflect.Type) *typeConfig {
	if m := configCache.Load(); m != nil {
		if c, ok := (*m)[t]; ok {
			return c
		}
	}
	configMu.Lock()
	defer configMu.Unlock()
	if m := configCache.Load(); m != nil {
		if c, ok := (*m)[t]; ok {
			return c
		}
	}
	cfg := buildConfig(t)
	old := configCache.Load()
	size := 0
	if old != nil {
		size = len(*old)
	}
	next := make(map[reflect.Type]*typeConfig, size+1)
	if old != nil {
		for k, v := range *old {
			next[k] = v
		}
	}
	next[t] = cfg
	configCache.Store(&next)
	return cfg
}

// buildConfig 构建某类型的脱敏配置。
func buildConfig(t reflect.Type) *typeConfig {
	cfg := &typeConfig{}
	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue // 未导出字段跳过
		}
		tag := sf.Tag.Get("mask")
		fc := fieldConfig{name: sf.Name, isNested: elemStruct(sf.Type)}
		if tag == "" {
			if fc.isNested {
				cfg.fields = append(cfg.fields, fc)
			}
			continue
		}
		if tag == "-" {
			continue
		}
		// 通用格式：mask:"3,4"（保留前 3 后 4）
		if strings.Contains(tag, ",") {
			head, tail, ok := parseKeep(tag)
			if !ok {
				cfg.errs = append(cfg.errs, maskError{sf.Name, tag})
				continue
			}
			fc.format = keepBoth(head, tail)
		} else {
			fn := formatFor(tag)
			if fn == nil {
				cfg.errs = append(cfg.errs, maskError{sf.Name, tag})
				continue
			}
			fc.format = fn
		}
		cfg.fields = append(cfg.fields, fc)
	}
	return cfg
}

// parseKeep 解析 "3,4" 形式的通用保留参数。
func parseKeep(tag string) (head, tail int, ok bool) {
	parts := strings.Split(tag, ",")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	ta, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || h < 0 || ta < 0 {
		return 0, 0, false
	}
	return h, ta, true
}

// elemStruct 判断类型（解引用指针/接口，或切片/数组的元素）最终是否是结构体。
func elemStruct(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr || t.Kind() == reflect.Interface {
		if t.Kind() == reflect.Interface {
			return false
		}
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return true
	}
	if t.Kind() == reflect.Slice || t.Kind() == reflect.Array {
		et := t.Elem()
		for et.Kind() == reflect.Ptr || et.Kind() == reflect.Interface {
			if et.Kind() == reflect.Interface {
				return false
			}
			et = et.Elem()
		}
		return et.Kind() == reflect.Struct
	}
	return false
}

// Mask 就地脱敏结构体（修改 v 指向的字段值）。v 必须是结构体指针，或结构体切片指针。
// 配置错误（未注册格式）返回错误；格式名拼错会在首次调用时暴露。
func Mask(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrNotPointer
	}
	elem := rv.Elem()
	var errs []string
	seen := make(map[any]struct{})
	switch elem.Kind() {
	case reflect.Struct:
		maskStruct(elem, "", &errs, seen)
	case reflect.Slice, reflect.Array:
		for i := 0; i < elem.Len(); i++ {
			ev := elem.Index(i)
			for ev.Kind() == reflect.Ptr || ev.Kind() == reflect.Interface {
				if ev.IsNil() {
					break
				}
				ev = ev.Elem()
			}
			if ev.Kind() == reflect.Struct {
				maskStruct(ev, fmt.Sprintf("[%d].", i), &errs, seen)
			}
		}
	default:
		return ErrNotStruct
	}
	if len(errs) > 0 {
		return fmt.Errorf("mask: %s", strings.Join(errs, "; "))
	}
	return nil
}

// MaskOf 是 Mask 的泛型入口：编译期保证 *T。
func MaskOf[T any](v *T) error {
	return Mask(v)
}

// maskStruct 递归脱敏结构体，path 用于错误消息。
func maskStruct(rv reflect.Value, path string, errs *[]string, seen map[any]struct{}) {
	if rv.CanAddr() {
		key := rv.Addr().Interface()
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
	}
	cfg := getConfig(rv.Type())
	if cfg == nil {
		return
	}
	for _, me := range cfg.errs {
		*errs = append(*errs, fmt.Sprintf("字段 %s: 格式 %q 未注册", path+me.field, me.name))
	}
	for _, fc := range cfg.fields {
		maskField(rv.FieldByName(fc.name), fc, path, errs, seen)
	}
}

// maskField 脱敏单个字段：string 应用格式，嵌套递归。
func maskField(fv reflect.Value, fc fieldConfig, path string, errs *[]string, seen map[any]struct{}) {
	fieldName := path + fc.name

	if fc.format != nil {
		if fv.Kind() == reflect.String {
			fv.SetString(fc.format(fv.String()))
			return // string 字段不递归（无嵌套可能）
		}
		// 非 string 字段带 mask tag：忽略格式，仅当嵌套时继续递归。
	}

	if !fc.isNested || !fv.IsValid() {
		return
	}
	switch fv.Kind() {
	case reflect.Struct:
		maskStruct(fv, fieldName+".", errs, seen)

	case reflect.Ptr, reflect.Interface:
		if fv.IsNil() {
			return
		}
		elem := fv.Elem()
		for elem.Kind() == reflect.Ptr || elem.Kind() == reflect.Interface {
			if elem.IsNil() {
				return
			}
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Struct {
			maskStruct(elem, fieldName+".", errs, seen)
		}

	case reflect.Slice, reflect.Array:
		for i := 0; i < fv.Len(); i++ {
			ev := fv.Index(i)
			for ev.Kind() == reflect.Ptr || ev.Kind() == reflect.Interface {
				if ev.IsNil() {
					break
				}
				ev = ev.Elem()
			}
			if ev.Kind() == reflect.Struct {
				maskStruct(ev, fmt.Sprintf("%s[%d].", fieldName, i), errs, seen)
			}
		}
	}
}
