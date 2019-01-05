package main

import (
	"bufio"
	"io"
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
		log.Println("Starting reader")
		go readChannel(device.SerialPort)
		log.Println("Requesting temperature")
		_, err := device.SerialPort.Write([]byte("M105\n"))
		log.Println("Request sent")
		if err != nil {
			log.Println(err)
		}
	}

	file, err := os.Open("./hook.gcode")
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		//fmt.Print(scanner.Text())
	}
	for {
	}
}

func readChannel(r io.Reader) {
	for {
		buf := make([]byte, 128)
		r.Read(buf)
		log.Println(string(buf))
	}
}
