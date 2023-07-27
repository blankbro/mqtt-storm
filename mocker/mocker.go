package mocker

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Mocker interface {
	NewClientOptions() *mqtt.ClientOptions
	Sub(client mqtt.Client) error
	Pub(client mqtt.Client, param interface{})
}
