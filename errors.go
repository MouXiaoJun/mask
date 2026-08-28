package mask

import "errors"

var (
	// ErrNotPointer 传入的不是非 nil 指针。
	ErrNotPointer = errors.New("mask: value must be a non-nil pointer")
	// ErrNotStruct 指针指向的不是结构体。
	ErrNotStruct = errors.New("mask: value must be a pointer to struct")
)
