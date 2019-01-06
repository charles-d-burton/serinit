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
		readerChan := make(chan string, 1)
		writerChan := writeChannel(device.SerialPort)
		log.Println("Starting reader")
		go readChannel(device.SerialPort, readerChan)
		//go requestTemps(writerChan)
		//go requestTemps(device.SerialPort)
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
		if value == "ok" {
			return true
		} else {
			return waitForOk(r)
		}
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
	buf := make(chan string, 10)
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

/*func readChannel(deviceAddr string, reader chan string) {
	t, err := follower.New(deviceAddr, follower.Config{
		Whence: io.SeekEnd,
		Offset: 0,
		Reopen: true,
	})
	if err != nil {
		log.Fatal(err)
	}
	for line := range t.Lines() {
		reader <- line.String()
	}
}*/

func readChannel(r io.Reader, reader chan string) {
	buf := make([]byte, 128)
	for {
		len, err := r.Read(buf)
		log.Println("Got message!")
		if err != nil {
			log.Fatal(err)
		}
		log.Println(string(buf[0:len]))
		reader <- string(buf[0:len])
	}
}

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
	command := string([]byte(line)[0:idx])
	return command + "\n"
}
