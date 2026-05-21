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
	asaAddr := os.Getenv("ASA_ADDR")
	if asaAddr == "" {
		asaAddr = "fake-asa:23"
	}

	srv, err := testhelper.NewSSHServerOnPort(testhelper.CiscoSwitchConfig(), 22)
	if err != nil {
		log.Fatalf("start: %v", err)
	}
	srv.TelnetTargets["10.1.31.251"] = asaAddr

	log.Printf("fake-switch listening on :22 (telnet target 10.1.31.251 → %s)", asaAddr)
	fmt.Println("user=admin pass=admin123 enable=enable123")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	srv.Close()
}
