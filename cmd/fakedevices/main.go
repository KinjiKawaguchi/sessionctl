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
	// Start fake ASA (telnet on port 2323)
	asaSrv, err := testhelper.NewTelnetServerOnPort(testhelper.ASAConfig(), 2323)
	if err != nil {
		log.Fatalf("start ASA: %v", err)
	}
	log.Printf("fake-asa (telnet): %s  user=admin pass=admin123", asaSrv.Addr())

	// Start fake switch (SSH on port 2222) with telnet target to ASA
	switchSrv, err := testhelper.NewSSHServerOnPort(testhelper.CiscoSwitchConfig(), 2222)
	if err != nil {
		log.Fatalf("start switch: %v", err)
	}
	switchSrv.TelnetTargets["10.1.31.251"] = asaSrv.Addr()
	log.Printf("fake-switch (ssh):  %s  user=admin pass=admin123 enable=enable123", switchSrv.Addr())

	// Start fake RTX (SSH on port 2223)
	rtxSrv, err := testhelper.NewSSHServerOnPort(testhelper.YamahaRTXConfig(), 2223)
	if err != nil {
		log.Fatalf("start RTX: %v", err)
	}
	log.Printf("fake-rtx (ssh):     %s  user=admin pass=admin123", rtxSrv.Addr())

	fmt.Println("\n--- Fake devices running. Press Ctrl+C to stop. ---")
	fmt.Println()
	fmt.Println("Test commands:")
	fmt.Println("  ssh -p 2222 admin@127.0.0.1       # Cisco Switch")
	fmt.Println("  ssh -p 2223 admin@127.0.0.1       # Yamaha RTX")
	fmt.Println("  telnet 127.0.0.1 2323              # Cisco ASA")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	switchSrv.Close()
	asaSrv.Close()
	rtxSrv.Close()
	fmt.Println("\nstopped.")
}
