package main

import (
	"log"

	"github.com/charles-d-burton/serinit"
)

func main() {
	devices, err := serinit.GetDevices()
	if err != nil {
		log.Println(err)
	}

	for _, device := range devices {
		log.Println("Requesting temperature")
		_, err := device.SerialPort.Write([]byte("M105\n"))
		if err != nil {
			log.Println(err)
		}
		buf := make([]byte, 128)
		device.SerialPort.Read(buf)
		log.Println(string(buf))
	}
}
