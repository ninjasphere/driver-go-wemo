package main

import (
	"fmt"
	"os"
	"os/signal"
)

func main() {

	_, err := NewWemoDriver()

	if err != nil {
		log.Fatalf("Failed to create Wemo driver: %s", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	// Block until a signal is received.
	s := <-c
	fmt.Println("Got signal:", s)

}
