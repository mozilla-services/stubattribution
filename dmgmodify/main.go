package main

import (
	"log"
	"os"

	"github.com/mozilla-services/stubattribution/dmglib"
	"github.com/mozilla-services/stubattribution/dmgmodify/dmgmodify"
)

func main() {
	if len(os.Args) != 5 {
		log.Fatalf("Usage: %s input.dmg output.dmg replacement\n", os.Args[0])
	}
	input, err := dmglib.OpenFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer input.Close()

	output, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer output.Close()

	dmgObj, err := input.Parse()
	if err != nil {
		log.Fatal(err)
	}

	err = dmgmodify.WriteAttributionCode(dmgObj, []byte(os.Args[3]))
	if err != nil {
		log.Fatal(err)
	}

	output.Write(dmgObj.Data)
}
