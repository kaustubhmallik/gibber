package main

import (
	"context"
	"gibber/service"
	"log"
	"time"
)

// details of the endpoint exposed
const (
	host = "0.0.0.0"
	port = "7000"
)

func main() {
	_, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	log.Fatal(service.StartServer(host, port, cancelFunc))
}
