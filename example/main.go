package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"

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
		value := scanner.Text()
		if strings.HasPrefix(value, ";") {
			log.Println("Comment: " + value)
		} else {
			devices[0].SerialPort.Write([]byte(value + "\n"))
		}
		//fmt.Print(scanner.Text())
	}
}

func readChannel(r io.Reader) {
	for {
		buf := make([]byte, 128)
		r.Read(buf)
		log.Println(string(buf))
	}
}
