package mymocker

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type MyMocker struct {
	Broker   string
	Username string
	Password string
}

func (myMocker *MyMocker) NewClientOptions() *mqtt.ClientOptions {
	options := mqtt.NewClientOptions()
	options.AddBroker(myMocker.Broker)
	options.SetUsername(myMocker.Username)
	options.SetPassword(myMocker.Password)
	options.SetClientID(strings.Replace(uuid.New().String(), "-", "", -1))
	options.SetConnectTimeout(30 * time.Second)
	options.SetCleanSession(true)
	options.SetAutoReconnect(false)
	return options
}

func (*MyMocker) SubStorm(client mqtt.Client) error {
	reader := client.OptionsReader()
	token := client.Subscribe(fmt.Sprintf("/test/sub/%s", reader.ClientID()), 0, func(client mqtt.Client, message mqtt.Message) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] subscribed %s -> %s", optionsReader.ClientID(), message.Topic(), string(message.Payload()))
	})

	if token.Wait() && token.Error() != nil {
		logrus.Panicf("Client[%s] connect fail, error: %s", reader.ClientID(), token.Error().Error())
		return token.Error()
	}

	return nil
}

func (*MyMocker) PubStorm(client mqtt.Client) {
	reader := client.OptionsReader()
	clientId := reader.ClientID()
	msgCount := 0
	for client.IsConnected() {
		msgCount++
		client.Publish(
			fmt.Sprintf("/test/sub/%s", clientId),
			0, false,
			fmt.Sprintf("hello %d", msgCount),
		)
		time.Sleep(3 * time.Second)
	}
}
