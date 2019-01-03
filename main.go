package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/distributed/sers"
)

func main() {
	devices, err := getSerialDevices()
	if err != nil {
		log.Println(err)
	}
	for _, device := range devices {
		sp, err := sers.Open(device)
		defer sp.Close()
		if err != nil {
			log.Println(err)
		}
		err = sp.SetMode(250000, 8, sers.N, 1, sers.NO_HANDSHAKE)
		err = sp.SetReadParams(1, 1)
		if err != nil {
			log.Println(err)
			return
		}
		err = initializeConnection(sp)
		if err != nil {
			log.Println(err)
		}

		defer sp.Close()

	}
}

func initializeConnection(r io.ReadCloser) error {
	ch := make(chan []byte, 5)
	defer close(ch)

	go func() error {
		buf := make([]byte, 128)
		data, err := readData(r, buf)
		if err != nil {
			return err
		}
		ch <- data
		return nil
	}()
	select {
	case data := <-ch:
		fmt.Println(string(data))
	case <-time.After(10 * time.Second):
		return errors.New("Timeout waiting for device")
	}

	return errors.New("Problem reading from device")
}

func readData(r io.ReadCloser, buf []byte) ([]byte, error) {
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
