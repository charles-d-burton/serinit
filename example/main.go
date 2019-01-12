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
	devices, err := serinit.AutoDiscoverDevices()
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
		err = device.Write([]byte("M105\n"))
		err = device.Write([]byte("M155 S2\n")) //Request a temperature status every 2 seconds
		if err != nil {
			log.Println(err)
		}
		commandQueue := commandQueue(file)
		for command := range commandQueue {
			fmt.Printf(command)
			device.Write([]byte(command))
			waitForOk(device.Reader)
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
func commandQueue(f *os.File) chan string {
	buf := make(chan string, 500)
	go func() {
		scanner := bufio.NewScanner(f)
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
		close(buf)
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
	} else if idx < 0 {
		return line + "\n"
	}
	return string([]byte(line)[0:idx]) + "\n"
}
