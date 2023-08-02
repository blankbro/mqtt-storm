package storm

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/mocker"
	"sync"
	"time"
)

type MqttStorm struct {
	sync.RWMutex
	MqttClientMap map[string]mqtt.Client
	Mocker        mocker.Mocker
	started       bool
}

func NewMqttStorm(mocker mocker.Mocker) *MqttStorm {
	return &MqttStorm{
		MqttClientMap: make(map[string]mqtt.Client),
		Mocker:        mocker,
	}
}

func (ms *MqttStorm) Shutdown() {
	logrus.Infof("shutdown storm start")
	mqttClientSize := len(ms.MqttClientMap)
	for _, client := range ms.MqttClientMap {
		if client.IsConnected() {
			client.Disconnect(5_000)
		}
	}
	ms.started = false
	logrus.Infof("shutdown storm finish(client size: %d)", mqttClientSize)
}

func (ms *MqttStorm) Run(clientCount uint64) {
	if ms.started {
		return
	}
	ms.started = true
	go func() {
		lastClientSize := 0
		for ms.started {
			currClientSize := len(ms.MqttClientMap)
			if currClientSize != lastClientSize {
				logrus.Infof("clientSize: %d", len(ms.MqttClientMap))
				lastClientSize = currClientSize
			}
			time.Sleep(1 * time.Second)
		}
	}()

	addClientErr := ms.AddClientByTargetCount(clientCount)
	if addClientErr != nil {
		logrus.Errorf("AddClientByTargetCount error: %s", addClientErr.Error())
	}
}

var CountLessZero = fmt.Errorf("count <= 0")

func (ms *MqttStorm) AddClientByTargetCount(targetCount uint64) error {
	if targetCount <= 0 {
		return CountLessZero
	}

	ms.Lock()
	defer ms.Unlock()

	for currCount := uint64(len(ms.MqttClientMap)); currCount < targetCount; currCount++ {
		mqttClient, connectToken, err := ms.newMqttClient()
		if err != nil {
			return err
		}

		optionsReader := mqttClient.OptionsReader()
		clientId := optionsReader.ClientID()
		ms.MqttClientMap[clientId] = mqttClient

		lastClientId := ""
		if currCount == targetCount-1 {
			lastClientId = clientId
		}

		go func() {
			if connectToken.Wait() && connectToken.Error() != nil {
				ms.Lock()
				defer ms.Unlock()
				logrus.Errorf("mqttClient[%s] conn err: %s", clientId, connectToken.Error().Error())
				delete(ms.MqttClientMap, clientId)
			}

			if lastClientId != "" {
				logrus.Infof("AddClientByTargetCount(%d) finish", targetCount)
			}

		}()
	}

	return nil
}

func (ms *MqttStorm) RemoveClientByTargetCount(targetCount uint64) {
	if targetCount < 0 {
		targetCount = 0
	}

	currCount := uint64(len(ms.MqttClientMap))
	if currCount <= targetCount {
		return
	}

	ms.Lock()
	defer ms.Unlock()

	for clientId := range ms.MqttClientMap {
		if currCount <= targetCount {
			break
		}
		client, ok := ms.MqttClientMap[clientId]
		if ok {
			delete(ms.MqttClientMap, clientId)
			client.Disconnect(5_000)
			logrus.Debugf("Client[%s] disconnected", clientId)
			currCount--
		}
	}
}

func (ms *MqttStorm) newMqttClient() (mqtt.Client, mqtt.Token, error) {
	options := ms.Mocker.NewClientOptions()
	_, exist := ms.MqttClientMap[options.ClientID]
	retryCount := 0
	for exist {
		if retryCount >= 10 {
			logrus.Warnf("重试%d次，仍然没有创建成功", retryCount)
			return nil, nil, fmt.Errorf("重试%d次，仍然没有创建成功", retryCount)
		}
		retryCount++
		logrus.Warnf("创建新客户端时clientId[%s]与已有的客户端重复，开始第%d次重试", options.ClientID, retryCount)
		options = ms.Mocker.NewClientOptions()
		_, exist = ms.MqttClientMap[options.ClientID]
	}

	// 不允许自动重连。一旦有客户端断开连接说明已经到达瓶颈
	options.SetAutoReconnect(false)

	// 包装 DefaultPublishHandler
	publishHandler := options.DefaultPublishHandler
	options.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		optionsReader := client.OptionsReader()
		logrus.Infof("Client[%s] published %s -> %s", optionsReader.ClientID(), msg.Topic(), msg.Payload())
		if publishHandler != nil {
			publishHandler(client, msg)
		}
	})

	// 包装 OnConnectHandler
	connectHandler := options.OnConnect
	options.SetOnConnectHandler(func(client mqtt.Client) {
		optionsReader := client.OptionsReader()
		logrus.Debugf("Client[%s] connected", optionsReader.ClientID())
		if connectHandler != nil {
			connectHandler(client)
		}
	})

	// 包装 ConnectionLostHandler
	connectionLostHandler := options.OnConnectionLost
	options.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		ms.Lock()
		defer ms.Unlock()

		optionsReader := client.OptionsReader()
		logrus.Warnf("Client[%s] connection lost, error: %s", optionsReader.ClientID(), err.Error())
		delete(ms.MqttClientMap, optionsReader.ClientID())
		if connectionLostHandler != nil {
			connectionLostHandler(client, err)
		}
	})

	// 包装 ReconnectingHandler
	reconnectHandler := options.OnReconnecting
	options.SetReconnectingHandler(func(client mqtt.Client, options *mqtt.ClientOptions) {
		logrus.Infof("Client[%s] reconnecting", options.ClientID)
		if reconnectHandler != nil {
			reconnectHandler(client, options)
		}
	})

	client := mqtt.NewClient(options)
	token := client.Connect()
	return client, token, nil
}

func (ms *MqttStorm) SubStorm() (int32, int32, error) {
	successCount := int32(0)
	totalCount := int32(len(ms.MqttClientMap))
	for _, client := range ms.MqttClientMap {
		err := ms.Mocker.Sub(client)
		if err != nil {
			return successCount, totalCount, err
		}
		successCount++
	}
	return successCount, totalCount, nil
}

func (ms *MqttStorm) PubStorm(requestBodyBytes []byte) error {
	requestBody, err := ms.Mocker.ParsePubStormRequestBody(requestBodyBytes)
	if err != nil {
		return err
	}

	ms.Lock()
	for _, client := range ms.MqttClientMap {
		go ms.Mocker.Pub(client, requestBody)
	}
	ms.Unlock()

	return nil
}
