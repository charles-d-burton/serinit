## serinit

Connect to and initialize a serial connection on a Linux machine.  Tested on x86 and ARM(Raspbian)

### Description

This library finds all serial devices connected to your computer.  It will then attempt to iterate through
the devices and discover their baud rates by connecting to the device, setting the baud rate, and looking for
printable characters in the return from the device.

### Usage

```go
package main

import (
    "fmt"
)

func main() {
    devices, err := serinit.GetDevices()
	if err != nil {
		log.Println(err)
    }
    for device := range devices {
        log.Println(device.TTY)
    }
}
```