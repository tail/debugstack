package main

import "github.com/tail/debugstack"

func MyFunc(anArg int) {
	aLocal := 0xfeedf00d

	AnotherFunc(aLocal, 0xf00fc7c8)
}

func AnotherFunc(arg1 int, arg2 int) int {
	debugstack.Test()

	return arg1 + arg2
}

func main() {
	MyFunc(0xdeadbeef)
}
