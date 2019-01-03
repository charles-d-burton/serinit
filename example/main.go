package main

import (
	"log"

	"github.com/charles-d-burton/serinit"
)

func main() {
	_, err := serinit.GetDevices()
	if err != nil {
		log.Println(err)
	}
}
