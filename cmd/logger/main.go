package main

import (
	"os"
	"strconv"
	"time"
)

func main() {
	if value, ok := os.LookupEnv("SLEEP"); ok {
		println("Sleeping " + value + "s")
		s, _ := strconv.Atoi(value)
		time.Sleep(time.Duration(s) * time.Second)
	}

	println("bye")
}
