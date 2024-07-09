package reflectx

import (
	"reflect"
	"unsafe"
)

// Clone creates a deep copy of the given struct pointer.
func Clone(src any) any {
	if src == nil {
		panic("source cannot be nil")
	}

	srcVal := reflect.ValueOf(src)
	srcType := srcVal.Type()

	if srcType.Kind() != reflect.Ptr || srcType.Elem().Kind() != reflect.Struct {
		panic("source must be a pointer to a struct")
	}

	// Create a new instance of the struct type
	dstVal := reflect.New(srcType.Elem())
	copyStruct(dstVal.Elem(), srcVal.Elem())

	return dstVal.Interface()
}

// copyStruct copies data from src to dst struct
func copyStruct(dst, src reflect.Value) {
	for i := 0; i < src.NumField(); i++ {
		srcField := src.Field(i)
		dstField := dst.Field(i)

		if dstField.CanSet() {
			dstField.Set(srcField)
		} else if !srcField.IsNil() {
			copyUnexportedField(dstField, srcField)
		}
	}
}

// copyUnexportedField copies an unexported field from src to dst.
// It uses unsafe to bypass visibility restrictions.
func copyUnexportedField(dst, src reflect.Value) {
	if src.CanAddr() && !src.IsNil() {
		dstPtr := unsafe.Pointer(dst.UnsafeAddr())
		srcPtr := unsafe.Pointer(src.UnsafeAddr())
		memcopy(dstPtr, srcPtr, src.Type().Size())
	}
}

// memcopy performs a raw memory copy from src to dst of given size.
func memcopy(dst, src unsafe.Pointer, size uintptr) {
	for i := uintptr(0); i < size; i++ {
		*(*byte)(unsafe.Pointer(uintptr(dst) + i)) = *(*byte)(unsafe.Pointer(uintptr(src) + i))
	}
}
