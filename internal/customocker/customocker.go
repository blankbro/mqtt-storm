package customocker

import (
	"encoding/json"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Mocker struct {
	Broker          string
	Username        string
	Password        string
	pubMsgSucCount  int32
	pubMsgErrCounts sync.Map
	pubClientCount  int32
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
		timestamp, _ := strconv.ParseInt(string(message.Payload()), 10, 64)
		pubTime := time.UnixMilli(timestamp)
		logrus.Infof("耗时: %s", time.Now().Sub(pubTime).String())
	})

	if token.Wait() && token.Error() != nil {
		logrus.Panicf("Client[%s] connect fail, error: %s", reader.ClientID(), token.Error().Error())
		return token.Error()
	}

	return nil
}

type PubParam struct {
	// 持续推送时间
	PubDurationSecond float64
	// 推送频率
	PubFrequencyMs int64
	Qos            byte
}

func (m *Mocker) ParsePubParam(requestBodyBytes []byte) (interface{}, error) {
	var pubParam PubParam
	err := json.Unmarshal(requestBodyBytes, &pubParam)
	if err != nil {
		return nil, err
	}
	return pubParam, nil
}

func (m *Mocker) Pub(client mqtt.Client, param interface{}) {
	pubParam := param.(PubParam)
	pubFrequencyMs := pubParam.PubFrequencyMs

	atomic.AddInt32(&m.pubClientCount, 1)
	defer atomic.AddInt32(&m.pubClientCount, -1)

	reader := client.OptionsReader()
	clientId := reader.ClientID()
	startTime := time.Now()
	for client.IsConnected() && time.Now().Sub(startTime).Seconds() < pubParam.PubDurationSecond {
		// region 根据自己的需要修改这部分代码即可
		token := client.Publish(
			fmt.Sprintf("/test/sub/%s", clientId),
			pubParam.Qos, false,
			fmt.Sprintf("%d", time.Now().UnixMilli()),
		)
		// endregion
		if token.Wait(); token.Error() != nil {
			m.addPubMsgErrCount(token.Error().Error())
		} else {
			atomic.AddInt32(&m.pubMsgSucCount, 1)
		}
		time.Sleep(time.Duration(pubFrequencyMs) * time.Millisecond)
	}
}

func (m *Mocker) addPubMsgErrCount(errInfo string) {
	useOfClosedNetworkConnection := "use of closed network connection"
	writeConnectionResetByPeer := "write: connection reset by peer"
	writeBrokenPipe := "write: broken pipe"
	if strings.Contains(errInfo, useOfClosedNetworkConnection) {
		errInfo = useOfClosedNetworkConnection
	} else if strings.Contains(errInfo, writeConnectionResetByPeer) {
		errInfo = writeConnectionResetByPeer
	} else if strings.Contains(errInfo, writeBrokenPipe) {
		errInfo = writeBrokenPipe
	}

	oldCount, _ := m.pubMsgErrCounts.LoadOrStore(errInfo, int32(0))
	for !m.pubMsgErrCounts.CompareAndSwap(errInfo, oldCount, oldCount.(int32)+1) {
		oldCount, _ = m.pubMsgErrCounts.LoadOrStore(errInfo, int32(0))
	}
}

func (m *Mocker) ObservePub() {
	lastPubClientCount := int32(0)
	lastPubMsgSucCount := int32(0)
	lastPrintTime := time.Now()
	lastPubFailInfo := ""
	for true {
		// 成功发布消息统计信息
		currPubClientCount := m.pubClientCount
		currPubMsgSucCount := m.pubMsgSucCount
		currTime := time.Now()
		timePassed := currTime.Sub(lastPrintTime)
		pubCountIncrement := currPubMsgSucCount - lastPubMsgSucCount
		if currPubClientCount != lastPubClientCount || currPubMsgSucCount != lastPubMsgSucCount {
			pubSpeed := int64(pubCountIncrement) / int64(timePassed.Seconds())
			logrus.Infof("发布消息的客户端数量: %d, 发布成功的消息总量: %d, 发布成功的消息增量: %d, 发布成功的消息速率: %d/s",
				m.pubClientCount, currPubMsgSucCount, pubCountIncrement, pubSpeed)
			lastPrintTime = currTime
			lastPubMsgSucCount = currPubMsgSucCount
			lastPubClientCount = currPubClientCount
		}

		// 失败发布消息统计信息
		var errInfos []string
		m.pubMsgErrCounts.Range(func(errInfo, count any) bool {
			errInfos = append(errInfos, errInfo.(string))
			return true
		})
		sort.Strings(errInfos)
		currPubFailInfo := ""
		for _, errInfo := range errInfos {
			count, ok := m.pubMsgErrCounts.Load(errInfo)
			if ok {
				currPubFailInfo += fmt.Sprintf("\n%s ===> %d", errInfo, count.(int32))
			}
		}
		if currPubFailInfo != lastPubFailInfo {
			logrus.Infof("发布失败统计: %s", currPubFailInfo)
			lastPubFailInfo = currPubFailInfo
		}

		// 清空统计信息
		if currPubClientCount == 0 {
			m.pubMsgSucCount = 0
			lastPubMsgSucCount = 0
			m.pubMsgErrCounts.Range(func(errInfo, count any) bool {
				m.pubMsgErrCounts.Delete(errInfo)
				return true
			})
			lastPubFailInfo = ""
		}
		time.Sleep(1 * time.Second)
	}
}
