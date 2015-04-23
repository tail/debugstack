package main

import (
	"fmt"

	"github.com/tail/debugstack"
)

func MyFunc(anArg int) {
	aLocal := 0xfeedf00d

	AnotherFunc(aLocal, 0xf00fc7c8)
}

func AnotherFunc(arg1 int, arg2 int) int {
	fmt.Printf("name 0: %s\n", debugstack.GetParamsLocalsForCaller(0))
	fmt.Printf("name 1: %s\n", debugstack.GetParamsLocalsForCaller(1))

	pclntab := debugstack.GetPclntab()
	fmt.Printf("FPForCaller(0) = 0x%x\n", debugstack.FPForCaller(pclntab, 0))
	fmt.Printf("FPForCaller(1) = 0x%x\n", debugstack.FPForCaller(pclntab, 1))

	return arg1 + arg2
}

func main() {
	MyFunc(0xdeadbeef)
}
