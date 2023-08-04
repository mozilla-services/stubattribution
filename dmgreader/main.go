package main

import (
	"fmt"
	"log"

	"github.com/willdurand/go-dmg-reader/dmg"
)

func main() {
	file, err := dmg.OpenFile("/Users/william/Downloads/Firefox 110.0.dmg")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data, err := file.ParseXMLPropertyList()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(data)
}