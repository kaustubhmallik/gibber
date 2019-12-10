package main

import (
	"gibber/service"
	"log"
)

func main() {
	log.Fatal(service.StartServer())
}
