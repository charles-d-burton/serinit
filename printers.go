package serinit

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"
	"unicode"

	"github.com/distributed/sers"
)

var (
	bauds = []int{
		110,
		300,
		600,
		1200,
		2400,
		4800,
		9600,
		14400,
		19200,
		28800,
		38400,
		56000,
		57600,
		115200,
		128000,
		153600,
		230400,
		250000,
		256000,
		460800,
		921600,
	}
	commonBauds = []int{
		9600,
		19200,
		115200,
		250000,
	}
)

type SerialDevice struct {
	TTY        string          `json:"tty"`
	SerialPort sers.SerialPort `json:"-"`
	Baud       int             `json:"baud"`
}

func GetDevices() ([]SerialDevice, error) {
	var devices []SerialDevice
	discovered, err := getSerialDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range discovered {
		var device SerialDevice
		device.TTY = d
		found, err := device.findBaudRate()
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.New("Unable to determine baud rate")
		}
		fmt.Printf("Found working baud: %d", device.Baud)
		devices = append(devices, device)

	}
	return devices, nil
}

func (device *SerialDevice) findBaudRate() (bool, error) {
	fmt.Println("Testing common bauds")
	for _, baud := range commonBauds {
		sp, err := sers.Open(device.TTY)
		if err != nil {
			return false, err
		}
		device.SerialPort = sp
		fmt.Printf("Setting baud to: %d\n", baud)
		device.Baud = baud
		found, err := testBaud(baud, device.SerialPort)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
		sp.Close()
	}
	fmt.Println("Common bauds failed, attempting more comprehensive list")
	for _, baud := range bauds {
		sp, err := sers.Open(device.TTY)
		if err != nil {
			return false, err
		}
		device.SerialPort = sp
		fmt.Printf("Setting baud to: %d\n", baud)
		device.Baud = baud
		found, err := testBaud(baud, device.SerialPort)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
		sp.Close()
	}
	return false, nil
}

func testBaud(baud int, sp sers.SerialPort) (bool, error) {
	err := sp.SetMode(baud, 8, sers.N, 1, sers.NO_HANDSHAKE)
	duration := 2 * time.Second
	time.Sleep(duration)
	if err != nil {
		return false, err
	}
	err = sp.SetReadParams(1, 1)
	if err != nil {
		return false, err
	}
	read, err := readUntilTimeout(sp)
	if err != nil {
		return false, err
	}
	if read {
		return true, nil
	}
	fmt.Printf("Baud %d failed\n", baud)
	return false, nil
}

func readUntilTimeout(r io.ReadCloser) (bool, error) {
	dataChan := make(chan []byte, 1)
	errorChan := make(chan error, 1)
	defer close(dataChan)
	defer close(errorChan)
	workingBaud := false
	go func() {
		data, err := readData(r)
		if err != nil {
			errorChan <- err
			return
		}
		dataChan <- data
	}()
	select {
	case data := <-dataChan:
		if isPrintable(string(data)) {
			workingBaud = true
			fmt.Print(string(data))
			for co := range dataChan {
				fmt.Print(string(co))
			}
			return true, nil
		} else {
			fmt.Println("Characters not printable!")
			return false, nil
		}

	case err := <-errorChan:
		return false, err
	case <-time.After(5 * time.Second):
		fmt.Println("Timeout of 5 seconds reached")
		return workingBaud, nil
	}
}

func readData(r io.ReadCloser) ([]byte, error) {
	buf := make([]byte, 128)
	_, err := r.Read(buf)
	return buf, err
}

//Retrieve the absolute path for serial devices
func getSerialDevices() ([]string, error) {
	//log.Println("getting serial devices")
	devices, err := ioutil.ReadDir("/dev/serial/by-id")
	if err != nil {
		log.Println(err)
		return nil, err
	}
	deviceList := make([]string, len(devices))
	for index, deviceLink := range devices {
		//log.Println("Found device: ", deviceLink.Name())
		abs, err := filepath.EvalSymlinks("/dev/serial/by-id/" + deviceLink.Name())
		//log.Print("Absolute Device: ")
		//log.Println(abs)
		deviceList[index] = abs
		if err != nil {
			log.Println(err)
			return nil, err
		}
	}
	return deviceList, nil
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			fmt.Println("FOUND NON PRINTABLE CHAR")
			fmt.Println(r)
			return false
		}
	}
	return true
}
