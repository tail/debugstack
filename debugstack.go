package debugstack

import (
	"debug/dwarf"
	"debug/elf"
	"fmt"
	"log"
	"runtime"
	"unsafe"

	"github.com/kardianos/osext"
	"github.com/tail/debugstack/realfunc"
)

func GetStackPointer() uintptr

// From https://bitbucket.org/sheran_gunasekera/leb128/
func DecodeSignedLEB128(value []byte) int32 {
	var result int32
	var ctr uint
	var cur byte = 0x80
	var signBits int32 = -1
	for (cur&0x80 == 0x80) && ctr < 5 {
		cur = value[ctr] & 0xff
		result += int32((cur & 0x7f)) << (ctr * 7)
		signBits <<= 7
		ctr++
	}
	if ((signBits >> 1) & result) != 0 {
		result += signBits
	}
	return result
}

func GetPclntab() []byte {
	executable, err := osext.Executable()
	if err != nil {
		log.Fatal(err)
	}

	file, err := elf.Open(executable)
	if err != nil {
		log.Fatal(err)
	}

	pclndat, err := file.Section(".gopclntab").Data()
	if err != nil {
		log.Fatal(err)
	}

	return pclndat
}

type ParamsLocals struct {
	name     string
	location int32
	kind     string
	value    int // HACK: need to make this an interface{} later.
}

func GetDwarfParamsLocals(funcname string) []*ParamsLocals {
	// TODO: Made up size.
	paramsLocals := make([]*ParamsLocals, 0, 16)

	// TODO: This is all one big hack right now.  We duplicate work by having
	// to re-open the executable and parse the DWARF info each time this is
	// called.
	executable, err := osext.Executable()
	if err != nil {
		log.Fatal(err)
	}

	file, err := elf.Open(executable)
	if err != nil {
		log.Fatal(err)
	}

	dwarfData, err := file.DWARF()
	if err != nil {
		log.Fatal(err)
	}

	dwarfReader := dwarfData.Reader()

	var entry *dwarf.Entry
	var entryName string
	var ok bool

outer:
	for {
		entry, err = dwarfReader.Next()
		if err != nil {
			log.Fatal(err)
		}
		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagSubprogram {
			entryName, ok = entry.Val(dwarf.AttrName).(string)
			if !ok {
				continue
			}
			if entryName != funcname {
				dwarfReader.SkipChildren()
			} else {
				log.Printf("DEBUG DWARF: Field name = %s\n", entryName)

				// TODO: holy indentation
				for {
					entry, err = dwarfReader.Next()
					if err != nil {
						log.Fatal(err)
					}
					if entry == nil {
						break outer
					}
					if entry.Tag == 0 {
						break outer
					}

					// Get AttrName / AttrLocation
					if entry.Tag == dwarf.TagVariable || entry.Tag == dwarf.TagFormalParameter {
						var location int32

						fmt.Printf("  Child Entry: %#v\n", entry)

						// TODO: error handling here is non-existant
						locationRaw := entry.Val(dwarf.AttrLocation).([]byte)
						if locationRaw[0] != 0x9c { // DW_OP_call_frame_cfa
							log.Fatalf("Unexpected opcode: %#v\n", locationRaw[0])
						}

						// TODO: Actually figure out how the location
						// expression is structured.  Just going off what it
						// looks like right now.
						if len(locationRaw) == 1 {
							location = 0
						} else {
							if locationRaw[1] != 0x11 { // DW_OP_consts
								log.Fatalf("Unexpected opcode: %#v\n", locationRaw[1])
							}
							if locationRaw[len(locationRaw)-1] != 0x22 { // DW_OP_plus
								log.Fatalf("Unexpected opcode: %#v\n", locationRaw[len(locationRaw)-1])
							}
							location = DecodeSignedLEB128(locationRaw[2 : len(locationRaw)-1])
						}

						fmt.Printf("    AttrName = %s, location = %#d\n",
							entry.Val(dwarf.AttrName).(string), location)

						var kind string
						if entry.Tag == dwarf.TagVariable {
							kind = "variable"
						} else {
							kind = "parameter"
						}

						paramsLocals = append(paramsLocals, &ParamsLocals{
							entry.Val(dwarf.AttrName).(string),
							location,
							kind,
							0,
						})
					}
				}
			}
		}
	}

	return paramsLocals
}

// Returns the frame pointer for caller.  The skip argument is the number of
// frames to descend, where 0 is the frame of the caller of this function.
// TODO: Calling this function _may_ trigger resizing the stack which means
// the addresses would change.  Does that mean you should always call this
// function twice to be safe?
// TODO: This could probably be optimized by only getting the pc for the
// current caller and using the pcsp table alone to walk the stack.
func FPForCaller(pclntab []byte, skip int) uintptr {
	// The stack pointer starts inside of GetStackPointer().  Every iteration
	// while we walk the stack, we need to first add sizeof(uintptr) to "pop"
	// the PC off the stack from the function return.  Then we need to add the
	// decoded pcsp value for the program counter to get the stack pointer at
	// each caller.
	// See NOTE below.  We don't want to assign this until after we're done
	// calling all functions.
	var fp uintptr
	pointerSize := unsafe.Sizeof(uintptr(0))

	// caller(0) is _this_ function.  To behave like runtime.Caller, we just
	// walk the stack one more than requested so this frame just becomes a
	// black box.
	skip += 1

	// Walk the stack `skip` frames.
	for caller := 0; caller <= skip; caller += 1 {
		// Get the program counter for the current frame.
		pc, _, _, ok := runtime.Caller(caller)
		if !ok {
			log.Fatal(ok)
		}
		fn := realfunc.RealFuncForPC(pc)

		// NOTE: We want to call GetStackPointer() as late as possible because
		// any function call in here has the possibility of relocating the
		// stack.  GetStackPointer() does not use any stack space and should
		// be safe, however we should _NOT_ call any new functions after this
		// point.
		if caller == 0 {
			fp = GetStackPointer()

			// Manually unwind from GetStackPointer() (i.e. pop off the program
			// counter) since it won't be part of the loop below.  At this point,
			// "fp" points to the stack pointer where this function has returned
			// from GetStackPointer.
			fp += pointerSize
		}

		// Unwind stack to beginning of the function of the current caller.
		fp += uintptr(fn.Pcsp(pc, pclntab))
		// Pop off program counter.
		fp += pointerSize

		// log.Printf("DEBUG FPForCaller[%d]: fn = %#v, pcsp_val = %d, fp = %x, filename = %s, line = %d, args = %d, skip = %d\n",
		// 	caller, fn, pcsp_val, fp, filename, line, fn.args, skip)
	}
	return fp
}

// func GetParamsLocalsForCaller(skip int) []*ParamsLocals {
func GetParamsLocalsForCaller(skip int) string {
	// Get function name
	// TODO: This is duplicated in FPForCaller.
	pc, _, _, ok := runtime.Caller(skip)
	if !ok {
		log.Fatal(ok)
	}
	fn := realfunc.RealFuncForPC(pc)

	return fn.Name()
}
