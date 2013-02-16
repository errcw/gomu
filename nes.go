package main

import "fmt"

func main() {
	a := uint8(0x80)
	b := uint8(0xff)
	fmt.Println(uint16(a + b))
	fmt.Println(a + b)
}
