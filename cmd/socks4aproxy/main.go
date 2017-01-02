package main

import (
	"log"
	"time"

	"github.com/henkman/socks4a"
)

func main() {
	var srv socks4a.Server
	srv.ReadTimeout = time.Second * 10
	srv.MaxUserIdLength = 1024
	srv.MaxNameLength = 1024
	log.Fatal(srv.ListenAndServe(":1080"))
}
