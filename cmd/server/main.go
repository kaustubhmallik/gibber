package main

import (
	"context"
	"gibber/service"
	"log"
	"time"
)

const (
	Host = "127.0.0.1"
	Port = "7000"
)

func main() {
	_, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	log.Fatal(service.StartServer(Host, Port, cancelFunc))
}
