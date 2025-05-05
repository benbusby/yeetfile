//go:build windows

package lang

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// Detect language on Winbdows
func DetectSystemLanguage() string {

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getLocaleName := kernel32.NewProc("GetUserDefaultLocaleName")

	buf := make([]uint16, 85) // max size according to Microsoft docs
	_, _, _ = getLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))

	for i, v := range buf {
		if v == 0 {
			buf = buf[:i]
			break
		}
	}

	locale := string(utf16.Decode(buf))
	return parseLangCode(locale)
}
