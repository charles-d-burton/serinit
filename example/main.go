package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

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
		go requestTemps(device.SerialPort)
		//log.Println("Requesting temperature")
		//_, err := device.SerialPort.Write([]byte("M105\n"))
		//log.Println("Request sent")
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
			if strings.HasPrefix(value, ";") {
				log.Println("Comment: " + value)
			} else {
				log.Println("Sending Command: " + value)
				device.SerialPort.Write([]byte(value + "\n"))
				for {
					select {
					case retval := <-readerChan:
						if strings.Contains(retval, "ok") {
							log.Println("Breaking out of loop to send next command")
							break
						} else {
							_, err := device.SerialPort.Write([]byte("M105\n"))
							if err != nil {
								log.Println(err)
							}
						}
					}
					break
				}
			}
		}
	}
}

func requestTemps(r io.Writer) error {
	for {
		fmt.Println("Requesting temps M105")
		r.Write([]byte("M105\n"))
		time.Sleep(time.Second)
	}
}

func readChannel(r io.Reader, reader chan string) {
	for {
		//buf := make([]byte, 128)
		buf, err := ioutil.ReadAll(r)
		//_, err := r.Read(buf)
		fmt.Println("Read: " + string(buf))
		if err != nil {
			log.Fatal(err)
		}
		reader <- string(buf)
	}
}
