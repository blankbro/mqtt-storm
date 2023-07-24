package mocker

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Mocker interface {
	NewClientOptions() *mqtt.ClientOptions
	SubStorm(client mqtt.Client) error
	PubStorm(client mqtt.Client)
}
