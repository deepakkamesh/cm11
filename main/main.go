package main

import (
	"fmt"

	"github.com/deepakkamesh/cm11"
)

func main() {
	oc := make(chan cm11.ObjState)
	c := cm11.New("/dev/ttyUSB0", oc)
	if err := c.Init(); err != nil {
		fmt.Printf("Got Error %s", err)
	}

	for {
		v := <-oc
		fmt.Print("\nGot", v)
		fmt.Print("\n")
		if v.FunctionCode == "On" {
			c.SendCommand("C", "4", "On")
		}
		if v.FunctionCode == "Off" {
			c.SendCommand("C", "4", "Off")
		}

	}
}
