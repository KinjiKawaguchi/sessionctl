package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/KinjiKawaguchi/sessionctl/test/testhelper"
)

func main() {
	port := 23
	srv, err := testhelper.NewTelnetServerOnPort(testhelper.ASAConfig(), port)
	if err != nil {
		log.Fatalf("start: %v", err)
	}

	log.Printf("fake-asa listening on :%d", port)
	fmt.Println("user=admin pass=admin123")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.Close()
}
