package customocker

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Mocker struct {
	Broker   string
	Username string
	Password string
}

func (m *Mocker) NewClientOptions() *mqtt.ClientOptions {
	options := mqtt.NewClientOptions()
	options.AddBroker(m.Broker)
	options.SetUsername(m.Username)
	options.SetPassword(m.Password)
	options.SetClientID(strings.Replace(uuid.New().String(), "-", "", -1))
	options.SetConnectTimeout(30 * time.Second)
	options.SetCleanSession(true)
	options.SetAutoReconnect(false)
	return options
}

func (*Mocker) Sub(client mqtt.Client) error {
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

type PubParam struct {
	MsgCount        int64
	PushFrequencyMs int64
	Qos             byte
}

func (*Mocker) ParsePubStormRequestBody(requestBody []byte) (interface{}, error) {
	var pubParam PubParam
	err := json.Unmarshal(requestBody, &pubParam)
	if err != nil {
		fmt.Println("Failed to parse JSON:", err)
		return nil, err
	}

	return pubParam, nil
}

func (*Mocker) Pub(client mqtt.Client, param interface{}) {
	pubParam := param.(PubParam)
	msgCount := pubParam.MsgCount
	pushFrequencyMs := pubParam.PushFrequencyMs
	qos := pubParam.Qos

	reader := client.OptionsReader()
	clientId := reader.ClientID()
	for pubCount := int64(0); client.IsConnected() && pubCount <= msgCount; pubCount++ {
		token := client.Publish(
			fmt.Sprintf("/test/sub/%s", clientId),
			qos, false,
			fmt.Sprintf("hello %d", msgCount),
		)
		token.Wait()
		if token.Error() != nil {
			logrus.Errorf("publish error: %s", token.Error().Error())
		}
		time.Sleep(time.Duration(pushFrequencyMs) * time.Millisecond)
	}
}
