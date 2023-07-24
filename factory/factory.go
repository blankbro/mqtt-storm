package factory

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/mocker"
	"math"
	"sync"
	"time"
)

type MqttClientFactory struct {
	sync.RWMutex
	Broker         string
	Username       string
	Password       string
	MaxClientIdNum uint64
	MqttClientMap  map[string]mqtt.Client
	Mocker         mocker.Mocker
}

func NewMqttClientFactory(broker string, username string, password string, mocker mocker.Mocker) *MqttClientFactory {
	return &MqttClientFactory{
		Broker:        broker,
		Username:      username,
		Password:      password,
		MqttClientMap: make(map[string]mqtt.Client),
		Mocker:        mocker,
	}
}

func (factory *MqttClientFactory) Shutdown() {
	logrus.Infof("shutdown factory start")
	mqttClientSize := len(factory.MqttClientMap)
	disconnectedCount := 0
	for clientId, client := range factory.MqttClientMap {
		if client.IsConnected() {
			client.Disconnect(5_000)
		}
		disconnectedCount += 1
		logrus.Infof("%d/%d Client[%s] disconnected", disconnectedCount, mqttClientSize, clientId)
	}
	logrus.Infof("shutdown factory finish")
}

func (factory *MqttClientFactory) Run(clientNum uint64) {
	successCount, err := factory.AddClientByCount(clientNum)
	if err != nil {
		logrus.Warnf("成功初始化客户端百分比为: %d/%d, 终止原因: %s", successCount, clientNum, err.Error())
	} else {
		logrus.Infof("成功初始化客户端百分比为: %d/%d", successCount, clientNum)
	}
}

var CountLessZero = fmt.Errorf("count <= 0")

func (factory *MqttClientFactory) AddClientByCount(count uint64) (uint64, error) {
	if count <= 0 {
		return 0, CountLessZero
	}

	factory.Lock()
	defer factory.Unlock()

	for i := uint64(0); i < count; i++ {
		num := factory.MaxClientIdNum + 1
		clientId := fmt.Sprintf("client-%06d", num)
		mqttClient, err := factory.newMqttClient(clientId)
		if err != nil {
			return i + 1, err
		}
		factory.MaxClientIdNum = num
		factory.MqttClientMap[clientId] = mqttClient
		go factory.Mocker.SubStorm(mqttClient)
		go factory.Mocker.PubStorm(mqttClient)
	}

	return count, nil
}

func (factory *MqttClientFactory) RemoveClientByCount(count uint64) {
	count = uint64(math.Min(float64(factory.MaxClientIdNum), float64(count)))
	for clientId := range factory.MqttClientMap {
		if count <= 0 {
			break
		}
		client, ok := factory.MqttClientMap[clientId]
		if ok {
			delete(factory.MqttClientMap, clientId)
			client.Disconnect(5_000)
			logrus.Infof("Client[%s] disconnected", clientId)
			count--
		}
	}
}

func (factory *MqttClientFactory) newMqttClient(clientId string) (mqtt.Client, error) {
	options := mqtt.NewClientOptions()
	options.AddBroker(factory.Broker)
	options.SetUsername(factory.Username)
	options.SetPassword(factory.Password)
	options.SetClientID(clientId)
	options.SetConnectTimeout(30 * time.Second)
	options.SetCleanSession(true)
	options.SetAutoReconnect(false)
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
		logrus.Warnf("Client[%s] connection lost, error: %s", optionsReader.ClientID(), err.Error())
		delete(factory.MqttClientMap, optionsReader.ClientID())
	})
	options.SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
		logrus.Infof("Client[%s] reconnecting", options.ClientID)
	})

	client := mqtt.NewClient(options)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		logrus.Errorf("Client[%s] connect fail, error: %s", clientId, token.Error().Error())
		return nil, token.Error()
	}

	return client, nil
}
