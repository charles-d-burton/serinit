## serinit

Connect to and initialize a serial connection on a Linux machine.  Tested on x86 and ARM(Raspbian)

### Description

This library finds all serial devices connected to your computer.  It will then attempt to iterate through
the devices and discover their baud rates by connecting to the device, setting the baud rate, and looking for
printable characters in the return from the device.

### Dependencies
* Should be built on an ARM based system due to low level syscalls.  Building on x86 compiled for ARM is untested
* Go 1.11.x
* Run `go get -u`

### Usage

```go
package main

import (
    "fmt"
)

func main() {
    devices, err := serinit.AutoDiscoverDevices()
	if err != nil {
		log.Println(err)
    }
    for device := range devices {
        log.Println(device.TTY)
    }
}
```

For a more complete example see the example folder.