package mocker

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
	"time"
)

type Mocker struct {
	sync.RWMutex
	Broker      string
	Username    string
	Password    string
	ClientNum   uint64
	MqttClients map[string]mqtt.Client
}

func NewMocker(broker string, username string, password string) *Mocker {
	return &Mocker{
		Broker:      broker,
		Username:    username,
		Password:    password,
		MqttClients: make(map[string]mqtt.Client),
	}
}

func (m *Mocker) Shutdown() {
	logrus.Infof("shutdown mocker start")
	mqttClientSize := len(m.MqttClients)
	currentIndex := 0
	for clientId, client := range m.MqttClients {
		if client.IsConnected() {
			client.Disconnect(5_000)
		}
		logrus.Infof("%d/%d Client[%s] disconnected", currentIndex+1, mqttClientSize, clientId)
		currentIndex++
	}
	logrus.Infof("shutdown mocker finish")
}

func (m *Mocker) Run(clientNum uint64) {
	m.AddClient(clientNum)
}

func generateClientId(i uint64) string {
	return fmt.Sprintf("client-%06d", i)
}

var CountLessZero = fmt.Errorf("count <= 0")

func (m *Mocker) AddClient(count uint64) error {
	if count <= 0 {
		return CountLessZero
	}

	m.Lock()
	defer m.Unlock()

	for i := uint64(0); i < count; i++ {
		num := m.ClientNum + i
		clientId := generateClientId(num)
		mqttClient := newMqttClient(m.Broker, m.Username, m.Password, clientId)
		m.MqttClients[clientId] = mqttClient
		go pubStorm(mqttClient)
	}

	m.ClientNum = m.ClientNum + count
	return nil
}

func (m *Mocker) RemoveClient(count uint64) {
	m.Lock()
	defer m.Unlock()

	count = uint64(math.Min(float64(m.ClientNum), float64(count)))

	for i := uint64(0); i < count; i++ {
		m.ClientNum = m.ClientNum - 1
		clientId := generateClientId(m.ClientNum)
		client, ok := m.MqttClients[clientId]
		if ok {
			delete(m.MqttClients, clientId)
			client.Disconnect(5_000)
			logrus.Infof("Client[%s] disconnected", clientId)
		}
	}
}

func newMqttClient(broker string, username string, password string, clientId string) mqtt.Client {
	options := mqtt.NewClientOptions()
	options.AddBroker(broker)
	options.SetUsername(username)
	options.SetPassword(password)
	options.SetClientID(clientId)
	options.SetConnectTimeout(30 * time.Second)
	options.SetCleanSession(true)
	options.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] published %s -> %s", optionsReader.ClientID(), msg.Topic(), msg.Payload())
	})
	options.SetOnConnectHandler(func(client mqtt.Client) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] connected", optionsReader.ClientID())
	})
	options.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] connection lost, error: %s", optionsReader.ClientID(), err.Error())
	})
	options.SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
		logrus.Infof("Client[%s] reconnecting", options.ClientID)
	})

	client := mqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logrus.Panicf("Client[%s] connect fail, error: %s", clientId, token.Error().Error())
	}

	return client
}

// TODO
func subStorm(mqttClient mqtt.Client) {
	reader := mqttClient.OptionsReader()
	token := mqttClient.Subscribe(fmt.Sprintf("/test/sub/%s", reader.ClientID()), 0, func(client mqtt.Client, message mqtt.Message) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] subscribed %s -> %s", optionsReader.ClientID(), message.Topic(), string(message.Payload()))
	})

	if token.Wait() && token.Error() != nil {
		logrus.Panicf("Client[%s] connect fail, error: %s", reader.ClientID(), token.Error().Error())
	}
}

func pubStorm(mqttClient mqtt.Client) {
	for mqttClient.IsConnected() {

	}
}
