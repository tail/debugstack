package realfunc

import (
	"fmt"
	"log"
	"runtime"
	"unsafe"
)

// Func struct from runtime/runtime.h
// See: https://docs.google.com/document/u/0/d/1lyPIbmsYbXnpNj57a261hgOYVpNRcgydurVQIyZOz_o/pub.
//
// TODO: This should just be a C struct so that we don't have to recopy the
// data, however using cgo will cause go to use the OS-native assembler
// instead of go's assembler.  Perhaps this could be isolated to its own
// package to prevent that.
type RealFunc struct {
	entry   uintptr // start pc
	nameoff int32   // function name

	args  int32 // in/out arg size
	frame int32 // legacy frame size; use pcsp if possible

	pcsp   int32
	pcfile int32
	pcln   int32

	npcdata   int32
	nfuncdata int32
}

func (fn *RealFunc) Name() string {
	return "LOL"
}

func (fn *RealFunc) Pcsp(pc uintptr, pclntab []byte) int32 {
	// Decode the pcsp value from the pcltab for this pc.
	return pcvalue(unsafe.Pointer(fn), fn.pcsp, pc, pclntab)
}

func RealFuncForPC(pc uintptr) *RealFunc {
	// runtime.FuncForPC returns a pointer to the Func.
	fn := unsafe.Pointer(runtime.FuncForPC(pc))
	ptr := uintptr(fn) + unsafe.Sizeof(fn)

	log.Printf("DEBUG: fn = %p, ptr = %x\n", fn, ptr)

	return &RealFunc{
		*(*uintptr)(fn),
		*(*int32)(unsafe.Pointer(ptr)),

		*(*int32)(unsafe.Pointer(ptr + 4)),
		*(*int32)(unsafe.Pointer(ptr + 8)),

		*(*int32)(unsafe.Pointer(ptr + 12)),
		*(*int32)(unsafe.Pointer(ptr + 16)),
		*(*int32)(unsafe.Pointer(ptr + 20)),

		*(*int32)(unsafe.Pointer(ptr + 24)),
		*(*int32)(unsafe.Pointer(ptr + 28)),
	}
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
