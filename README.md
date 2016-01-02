# cm11_driver_golang
x10 CM11 driver package written in golang.
x10 protocol specification http://wanderingsamurai.net/electronics/cm11a-x10-protocol-document

## Dependency
This requires the serial package https://github.com/tarm/serial

## Usage 
```go
package main

import (
  "cm11"
  "fmt"
)

func main() {
  //Create a new channel to recieve reads from cm11
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
```
