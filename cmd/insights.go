package main

import (
	"fmt"
	"log"

	. "github.com/RedHatInsights/libinsights"
)

func main() {
	log.Println("Starting...")
	COLLECTORS_DIRECTORY = "./insights.d/"
	collectors, err := GetCollectors()
	if err != nil {
		log.Fatal(err)
	}

	// TODO Create a table with fields 'ID', 'Name', sorted by ID
	for _, collector := range collectors {
		fmt.Println("name:", collector.Meta.Name)
	}
	log.Println("Done.")
}
