typedef signed int int32;
// TODO: this is hard-coded for 64-bit right now.
typedef unsigned long long uintptr;

// Func struct from runtime/runtime.h
// See: https://docs.google.com/document/u/0/d/1lyPIbmsYbXnpNj57a261hgOYVpNRcgydurVQIyZOz_o/pub.
typedef struct
{
    uintptr entry;  // start pc
    int32   nameoff;// function name

    int32   args;   // in/out args size
    int32   frame;  // legacy frame size; use pcsp if possible

    int32   pcsp;
    int32   pcfile;
    int32   pcln;
    int32   npcdata;
    int32   nfuncdata;
} _func;
