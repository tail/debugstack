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
	RunMagic()

	return arg1 + arg2
}

func RunMagic() {
	var paramsLocals []*debugstack.ParamsLocals
	var paramLocal *debugstack.ParamsLocals

	fmt.Println("========== main.AnotherFunc ==========")
	paramsLocals = debugstack.GetParamsLocalsForCaller(1)
	for _, paramLocal = range paramsLocals {
		paramLocal.Print()
		fmt.Println()
	}

	fmt.Println("========== main.MyFunc ==========")
	paramsLocals = debugstack.GetParamsLocalsForCaller(2)
	for _, paramLocal = range paramsLocals {
		paramLocal.Print()
		fmt.Println()
	}
}

func main() {
	MyFunc(0xdeadbeef)
}
