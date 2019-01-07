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
		readerChan := readChannel(device.SerialPort)
		writerChan := writeChannel(device.SerialPort)

		//go requestTemps(writerChan)
		//go requestTemps(device.SerialPort)
		//log.Println("Requesting temperature")
		//_, err := device.SerialPort.Write([]byte("M105\n"))
		//log.Println("Request sent")
		writerChan <- "M105\n"
		writerChan <- "M105\n"
		writerChan <- "M105\n"
		time.Sleep(3 * time.Second)
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
			value := stripComments(scanner.Text())
			if value != "" {
				log.Println("Sending Command: " + value)
				writerChan <- value
				waitForOk(readerChan)
			} else {
				log.Println("Discarding comment")
			}

		}
	}
}

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

func requestTemps(w chan string) error {
	for {
		fmt.Println("Requesting temps M105")
		w <- "M105\n"
		time.Sleep(time.Second)
	}
}

func writeChannel(w io.Writer) chan string {
	buf := make(chan string, 1)
	go func() {
		for {
			select {
			case line := <-buf:
				log.Println("Got message to write: " + line)
				_, err := w.Write([]byte(line))
				log.Println("Message written!")
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
		fmt.Println("No comments in line")
		return line + "\n"
	}
	fmt.Println("Is command: " + line)
	return string([]byte(line)[0:idx]) + "\n"
}
