package serinit

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"
	"unicode"

	"github.com/distributed/sers"
	"github.com/tevino/abool"
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

const (
	//Parity enable Parity
	Parity = 1
	//NoParity disable Parity
	NoParity = 0
)

//SerialDevice container to represent the location of a serial device and return an io port to it
type SerialDevice struct {
	sync.Mutex
	Options    *SerialDeviceOptions
	DeviceName string `json:"device_name"`
	DeviceID   string `json:"device_id"`
	Reader     chan []byte
	ErrChan    chan error
	sp         sers.SerialPort
}

//SerialDeviceOptions
type SerialDeviceOptions struct {
	TTY       string `json:"tty"`
	Baud      int    `json:"baud"`
	DataBits  int    `json:"data_bits"`
	Parity    int    `json:"parity"`
	StopBits  int    `json:"stop_bits"`
	HandShake int    `json:"handshake"`
}

//New creates new device, sets up sane defaults for connection
func New(options *SerialDeviceOptions) (*SerialDevice, error) {
	if options == nil {
		return nil, errors.New("device options must be defined")
	}
	if options.TTY == "" {
		return nil, errors.New("device TTY undefined")
	}
	if options.Baud == 0 {
		options.Baud = 9600
	}
	if options.DataBits == 0 {
		options.DataBits = 8
	}
	device := &SerialDevice{
		Options:    options,
		DeviceName: "new-device",
	}
	valid, err := device.isBaudValid(options.Baud)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, errors.New("baud rate is not valid")
	}
	return device, nil
}

//AutoDiscoverDevices iterate through a list of serial devices and initialize them
func AutoDiscoverDevices() ([]*SerialDevice, error) {
	var devices []*SerialDevice
	discovered, err := getSerialDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range discovered {
		var device SerialDevice
		device.Options.TTY = d
		found, err := device.findBaudRate()
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, errors.New("Unable to determine baud rate")
		}
		fmt.Printf("Found working baud: %d\n", device.Options.Baud)
		device.initConnections()
		devices = append(devices, &device)

	}
	if len(devices) == 0 {
		return nil, errors.New("No devices found")
	}
	return devices, nil
}

//GetDeviceTTYs return a list of paths to serial devices
func GetDeviceTTYs() ([]string, error) {
	return getSerialDevices()
}

//ConnectDevice manually connects device bypassing auto discovery
func (device *SerialDevice) ConnectDevice() error {
	sp, err := sers.Open(device.Options.TTY)
	if err != nil {
		return err
	}
	if device.Options.Baud == 0 {
		return errors.New("Invalid Baud Rate")
	}
	if device.Options.DataBits == 0 {
		device.Options.DataBits = 8
	}
	fmt.Println("Connecting with: ", device)
	err = sp.SetMode(device.Options.Baud, device.Options.DataBits, device.Options.Parity, device.Options.StopBits, device.Options.HandShake)
	if err != nil {
		return err
	}
	device.sp = sp
	device.initConnections()
	select {
	case data := <-device.Reader:
		if !isPrintable(string(data)) {
			return errors.New("Data garbled, check your connection")
		}
	case err := <-device.ErrChan:
		return err
	}
	return nil
}

//Reset and reinitialize the connection
func (device *SerialDevice) Reset() error {
	device.Lock()
	defer device.Unlock()
	close(device.Reader)
	close(device.ErrChan)
	device.sp.Close()
	found, err := device.findBaudRate()
	if err != nil {
		return err
	}
	if !found {
		return errors.New("Unable to determine the baud rate for device")
	}
	return nil
}

//Close the connection to the device
func (device *SerialDevice) Close() error {
	return device.Close()
}

/*
 * Discover the baud rate, first connect to the most common.  Then try all the rest
 */
func (device *SerialDevice) findBaudRate() (bool, error) {
	fmt.Println("Testing common bauds")
	for _, baud := range commonBauds {
		valid, err := device.isBaudValid(baud)
		if err != nil {
			return false, err
		}
		if valid {
			return true, nil
		}
	}
	fmt.Println("Common bauds failed, attempting more comprehensive list")
	for _, baud := range bauds {
		valid, err := device.isBaudValid(baud)
		if err != nil {
			return false, err
		}
		if valid {
			return true, nil
		}
	}
	return false, nil
}

func (device *SerialDevice) isBaudValid(baud int) (bool, error) {
	sp, err := sers.Open(device.Options.TTY)
	if err != nil {
		return false, err
	}
	fmt.Printf("Setting baud to: %d\n", baud)
	device.Options.Baud = baud
	found, err := testBaud(baud, sp)
	if err != nil {
		sp.Close()
		return false, err
	}
	if found {
		device.sp = sp

		return true, nil
	} else {
		sp.Close()
		return false, nil
	}
}

//Create the connection then attempt to read from the serial port
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
	doneChan := make(chan bool, 1)   //To let the system know processing is done
	errorChan := make(chan error, 1) //Feed in an error if one is encountered
	defer close(errorChan)
	defer close(doneChan)
	workingBaud := abool.New()
	closing := abool.New()
	go func() {
		if closing.IsSet() {
			return
		}
		data, err := readData(r) //Read in some data
		if err != nil {
			errorChan <- err
		}
		if isPrintable(string(data)) { //Test if that data is garbage
			workingBaud.Set()
			fmt.Print(string(data))
			for {
				data, err := readData(r)
				if err != nil {
					errorChan <- err
				}
				if !closing.IsSet() {
					fmt.Print(string(data))
				} else {
					return
				}
			}
		} else {
			doneChan <- true //Data is garbage so exit
		}
	}()
	select {
	case <-doneChan:
		closing.Set()
		return false, nil
	case err := <-errorChan:
		closing.Set()
		return false, err
	case <-time.After(time.Second):
		fmt.Println("Timeout of 5 seconds reached")
		closing.Set()
		return workingBaud.IsSet(), nil
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

//test if the characters retrieved from the serial device are ASCII
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

//Setup IO to the device non-blocking writes
func (device *SerialDevice) initConnections() {
	device.Lock()
	defer device.Unlock()
	device.Reader = make(chan []byte, 100)
	device.ErrChan = make(chan error, 5)
	go func() {
		scanner := bufio.NewScanner(device.sp)
		for scanner.Scan() {
			device.Reader <- scanner.Bytes()
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Had an error")
			device.ErrChan <- err
			return
		}
	}()
}

//TODO: Rethink this, it's slow
//Write thread-safe function that takes in data and writes it to port
func (device *SerialDevice) Write(message []byte) error {
	device.Lock()
	defer device.Unlock()
	_, err := device.sp.Write(message)
	return err
}
