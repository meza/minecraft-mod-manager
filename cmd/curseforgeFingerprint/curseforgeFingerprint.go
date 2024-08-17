package curseforgeFingerprint

// #cgo CFLAGS: -Wall -pedantic
// #include <stdlib.h>
// #include "fingerprint.h"
import "C"
import (
	"strconv"
	"unsafe"
)

func GetFingerprintFor(filePath string) string {
	file := C.CString(filePath)
	defer C.free(unsafe.Pointer(file))

	hash := C.compute_hash(file)
	return strconv.FormatUint(uint64(hash), 10)
}
