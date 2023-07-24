package main

import (
	"flag"
	"github.com/timeway/mqtt-storm/internal/mymocker"
	"github.com/timeway/mqtt-storm/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	broker := flag.String("broker", "mqtt://127.0.0.1:1883", "URI(tcp://{ip}:{port}) of MQTT broker (required)")
	username := flag.String("username", "admin", "Username for connecting to the MQTT broker")
	password := flag.String("password", "admin", "Password for connecting to the MQTT broker")
	clientNum := flag.Uint64("c", 1, "Number of clients")
	flag.Parse()

	if *broker == "" {
		log.Printf("Invalid argument: broker is empty")
		return
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	mss := server.NewMqttStormServer(":8080", &mymocker.MyMocker{
		Broker:   *broker,
		Username: *username,
		Password: *password,
	}, *clientNum)
	mss.ListenAndServe()
}
