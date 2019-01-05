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
		readerChan := make(chan string, 1)
		log.Println("Starting reader")
		go readChannel(device.SerialPort, readerChan)
		log.Println("Requesting temperature")
		_, err := device.SerialPort.Write([]byte("M105\n"))
		log.Println("Request sent")
		if err != nil {
			log.Println(err)
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
			log.Println("Sending Command: " + value)
			if strings.HasPrefix(value, ";") {
				log.Println("Comment: " + value)
			} else {
				devices[0].SerialPort.Write([]byte(value + "\n"))
				retval := <-readerChan
				log.Println(retval)
				for !strings.Contains(retval, "ok") {
					retval = <-readerChan
					log.Println(retval)
				}
			}
			//fmt.Print(scanner.Text())
		}
	}

}

func readChannel(r io.Reader, reader chan string) {
	for {
		buf := make([]byte, 128)
		r.Read(buf)
		reader <- string(buf)
	}
}
