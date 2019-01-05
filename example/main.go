package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

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

	file, err := os.Open("./hook.gcode")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Print(scanner.Text())
	}
}
