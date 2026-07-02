package main

import (
	"fmt"
	"log"
	"os"
)

func main() {
	logger := log.New(os.Stdout, "[vine-agent] ", log.LstdFlags)
	logger.Println("starting...")

	fmt.Println("vine-agent is running")
}
