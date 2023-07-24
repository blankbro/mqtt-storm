package mocker

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Mocker interface {
	SubStorm(client mqtt.Client)
	PubStorm(client mqtt.Client)
}
