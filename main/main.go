package main

import (
	"fmt"

	"github.com/deepakkamesh/cm11"
)

func main() {
	oc := make(chan cm11.ObjState)
	errChan := make(chan error)
	c := cm11.New("/dev/ttyUSB0", oc, errChan)
	if err := c.Init(); err != nil {
		fmt.Printf("Got Error %s", err)
	}

	for {
		select {
		case v := <-oc:
			fmt.Print("\nGot", v)
			fmt.Print("\n")
			if v.FunctionCode == "On" {
				c.SendCommand("C", "4", "On")
			}
			if v.FunctionCode == "Off" {
				c.SendCommand("C", "4", "Off")
			}

		case err := <-errChan:
			fmt.Printf("Got Error %s", err)

		}

	}
}
