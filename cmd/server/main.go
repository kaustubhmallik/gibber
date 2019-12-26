package main

import (
	"gibber/service"
	"log"
)

const (
	Host = "127.0.0.1"
	Port = "7000"
)

func main() {
	log.Fatal(service.StartServer(Host, Port))
}
