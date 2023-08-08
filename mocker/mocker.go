package mocker

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Mocker interface {

	// NewClientOptions MQTT客户端属性
	NewClientOptions() *mqtt.ClientOptions

	// Sub 订阅逻辑
	Sub(client mqtt.Client) error

	// NewEmptyPubParam 创建一个空的 PubParam 变量，用于把 http.RequestBody 转换成 PubParam
	ParsePubParam(requestBodyBytes []byte) (interface{}, error)

	// Pub 发布逻辑
	Pub(client mqtt.Client, param interface{})

	// ObservePub 观察发布情况
	ObservePub()
}
