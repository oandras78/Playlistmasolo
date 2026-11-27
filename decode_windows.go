//go:build windows

package main

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

var procMultiByteToWideChar = syscall.NewLazyDLL("kernel32.dll").NewProc("MultiByteToWideChar")

// A Windows saját kódlap-táblázatait használja (1250, 852, 1252).
func decodeANSI(cp int, b []byte) string {
	if len(b) == 0 {
		return ""
	}
	n, _, _ := procMultiByteToWideChar.Call(uintptr(cp), 0,
		uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)), 0, 0)
	if n == 0 {
		return string(b)
	}
	buf := make([]uint16, n)
	procMultiByteToWideChar.Call(uintptr(cp), 0,
		uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)),
		uintptr(unsafe.Pointer(&buf[0])), n)
	return string(utf16.Decode(buf))
}
