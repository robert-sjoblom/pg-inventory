// Collector - Aggregates inventory data from sidecars and stores in timeseries database.
package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	fmt.Println("PostgreSQL Inventory Collector")
	// TODO: Load config
	log.Println("Collector service starting...")
	os.Exit(0)
}
