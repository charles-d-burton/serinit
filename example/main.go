package main

import (
	"bufio"
	"io"
	"log"
	"os"
	"strings"

	"github.com/charles-d-burton/serinit"
	"github.com/papertrail/go-tail/follower"
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
		go readChannel(device.TTY, readerChan)
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
				for {
					select {
					case retval := <-readerChan:
						if retval == "ok" {
							log.Println("Breaking out of loop to send next command")
							break
						} else {
							writerChan <- "M105\n"
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

/*func requestTemps(r io.Writer) error {
	for {
		fmt.Println("Requesting temps M105")
		r.Write([]byte("M105\n"))
		time.Sleep(time.Second)
	}
}*/

func writeChannel(w io.Writer) chan string {
	buf := make(chan string, 10)
	go func() {
		for {
			select {
			case line := <-buf:
				_, err := w.Write([]byte(line))
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}()
	return buf
}

func readChannel(deviceAddr string, reader chan string) {
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
}

func stripComments(line string) string {
	idx := strings.Index(line, ";")
	command := strings.TrimSpace(string([]byte(line)[0:idx]))
	return command + "\n"
}
