// The main reason this exists as a separate package is because importing "C"
// will cause assembly code to use the OS-native s assembler instead of go's.
// It also serves as a convenience to anyone else who wants public access to
// Go's internal Func struct.

package realfunc

/*
#include "realfunc.h"
*/
import "C"

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"
)

// This is just a wrapper around the  C struct so that we don't have to
// recopy the data
// TODO: Keep reference to pclntab so we don't have to keep passing it in to
// every method?
type RealFunc struct {
	opaque struct{}
}

func (fn *RealFunc) Raw() *C._func {
	return (*C._func)(unsafe.Pointer(fn))
}

func (fn *RealFunc) Name(pclntab []byte) string {
	return C.GoString((*C.char)(unsafe.Pointer(&pclntab[fn.Raw().nameoff])))
}

func (fn *RealFunc) Pcsp(pc uintptr, pclntab []byte) int32 {
	// Decode the pcsp value from the pcltab for this pc.
	return pcvalue(unsafe.Pointer(fn), int32(fn.Raw().pcsp), pc, pclntab)
}

func RealFuncForPC(pc uintptr) *RealFunc {
	fn := unsafe.Pointer(runtime.FuncForPC(pc))
	return (*RealFunc)(fn)
}

// offset is the offset to the desired pc-value table.
func pcvalue(f unsafe.Pointer, offset int32, targetpc uintptr, pclntable []byte) int32 {
	if offset == 0 {
		return -1
	}

	entry := *(*uintptr)(unsafe.Pointer(f)) // Func.entry [0]
	pc := entry
	fmt.Printf("  pcvalue entry pc: %v %v\n", entry, pc)

	encpctab := pclntable[offset:]
	// fmt.Printf("  encoded pcvalue table: %#v [...]\n", encpctab[:32])

	val := int32(-1)
	for {
		var ok bool
		encpctab, ok = step(encpctab, &pc, &val, pc == entry)
		if !ok {
			log.Fatal("PCVALUE STEP NOT OK")
		}

		if targetpc < pc {
			return val
		}
	}

	return val
}

// Ported from src/runtime/symtab.go
func step(encpctab []byte, pc *uintptr, val *int32, first bool) (newencpctab []byte, ok bool) {
	encpctab, uvdelta := readvarint(encpctab)
	if uvdelta == 0 && !first {
		return nil, false
	}
	if uvdelta&1 != 0 {
		uvdelta = ^(uvdelta >> 1)
	} else {
		uvdelta >>= 1
	}
	vdelta := int32(uvdelta)
	encpctab, pcdelta := readvarint(encpctab)

	// TODO: This is platform-dependent but is hardcoded for amd64 right now.
	_PCQuantum := uint32(1)

	*pc += uintptr(pcdelta * _PCQuantum)
	*val += vdelta
	return encpctab, true

}

// Ported from src/runtime/symtab.go
// readvarint reads a cvarint from encpctab.
func readvarint(encpctab []byte) (newencpctab []byte, val uint32) {
	var v, shift uint32
	for {
		b := encpctab[0]
		encpctab = encpctab[1:]
		v |= (uint32(b) & 0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
	}
	return encpctab, v
}
