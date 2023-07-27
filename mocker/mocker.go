package mocker

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Mocker interface {
	NewClientOptions() *mqtt.ClientOptions
	Sub(client mqtt.Client) error
	ParsePubStormRequestBody(requestBody []byte) (interface{}, error)
	Pub(client mqtt.Client, param interface{})
}
