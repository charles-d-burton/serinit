package main

import (
	"bufio"
	"fmt"
	"io"
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
		log.Println("Starting reader")
		file, err := os.Open("./hook.gcode")
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()
		readerChan := readChannel(device.SerialPort)
		_, err = device.SerialPort.Write([]byte("M105\n"))
		_, err = device.SerialPort.Write([]byte("M155 S2\n")) //Request a temperature status every 2 seconds
		if err != nil {
			log.Println(err)
		}
		commandQueue := commandQueue(file)
		select {
		case command := <-commandQueue:
			fmt.Printf(command)
			device.SerialPort.Write([]byte(command))
			waitForOk(readerChan)
		}
	}
}

//Recursively wait for a message that contains ok to wait to send next instruction
func waitForOk(r chan string) bool {
	select {
	case value := <-r:
		fmt.Println(value)
		if strings.Contains(value, "ok") {
			return true
		}
		return waitForOk(r)
	}
}

//Request temperature
func requestTemps(w chan string) error {
	for {
		fmt.Println("Requesting temps M105")
		w <- "M105\n"
		time.Sleep(time.Second)
	}
}

//Create a queue of commands ready to be issued
func commandQueue(r io.Reader) chan string {
	buf := make(chan string, 50)
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			value := stripComments(scanner.Text())
			if value != "" {
				buf <- value
			}
		}
		//TODO: Add safety here to cool down and move the print head
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}()
	return buf
}

//Channel to write data to the device
func writeChannel(w io.Writer) chan string {
	buf := make(chan string, 1)
	go func() {
		for {
			select {
			case line := <-buf:
				fmt.Printf(line)
				_, err := w.Write([]byte(line))
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()
	return buf
}

//Read from the io port forever
func readChannel(r io.Reader) chan string {
	readerChan := make(chan string, 5)
	scanner := bufio.NewScanner(r)
	go func() {
		for {
			for scanner.Scan() {
				readerChan <- scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		}
	}()
	return readerChan
}

//Strip comments from input lines prior to sending them to the printer
func stripComments(line string) string {
	line = strings.TrimSpace(line)
	idx := strings.Index(line, ";")
	if idx == 0 {
		fmt.Println("Is comment: " + line)
		return ""
	} else if idx == -1 {
		return line + "\n"
	}
	return string([]byte(line)[0:idx]) + "\n"
}
