package mymocker

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"time"
)

type MyMocker struct {
}

func (*MyMocker) SubStorm(client mqtt.Client) {
	reader := client.OptionsReader()
	token := client.Subscribe(fmt.Sprintf("/test/sub/%s", reader.ClientID()), 0, func(client mqtt.Client, message mqtt.Message) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] subscribed %s -> %s", optionsReader.ClientID(), message.Topic(), string(message.Payload()))
	})

	if token.Wait() && token.Error() != nil {
		logrus.Panicf("Client[%s] connect fail, error: %s", reader.ClientID(), token.Error().Error())
	}
}

func (*MyMocker) PubStorm(client mqtt.Client) {
	i := 0
	for client.IsConnected() {
		reader := client.OptionsReader()
		client.Publish(
			fmt.Sprintf("/test/sub/%s", reader.ClientID()),
			0, false,
			fmt.Sprintf("hello %d", i),
		)
		time.After(3 * time.Second)
	}
}
