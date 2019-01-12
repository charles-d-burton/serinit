package main

import (
	"bufio"
	"fmt"
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
		file, err := os.Open("./hook.gcode")
		if err != nil {
			log.Println(err)
			return
		}
		defer file.Close()
		readerChan := device.Reader
		defer close(readerChan)
		err = device.Write([]byte("M1055\n"))
		err = device.Write([]byte("M155 S2\n"))
		//_, err = device.SerialPort.Write([]byte("M105\n"))
		//_, err = device.SerialPort.Write([]byte("M155 S2\n")) //Request a temperature status every 2 seconds
		if err != nil {
			log.Println(err)
		}
		finished := make(chan bool, 1)
		defer close(finished)
		commandQueue := commandQueue(file, finished)
		defer close(commandQueue)
		for {
			done := false
			select {
			case command := <-commandQueue:
				pending := len(commandQueue)
				if pending == 0 && done {
					return
				}
				fmt.Printf(command)
				device.Write([]byte(command))
				waitForOk(readerChan)
			case done = <-finished:
				fmt.Println("Finished processing file")
			}
		}
	}
}

//Recursively wait for a message that contains ok to wait to send next instruction
func waitForOk(r chan []byte) bool {
	for {
		select {
		case value := <-r:
			fmt.Println(string(value))
			if strings.Contains(string(value), "ok") {
				return true
			}
		}
	}
}

//Create a queue of commands ready to be issued
func commandQueue(r io.Reader, done chan bool) chan string {
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
		done <- true
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
