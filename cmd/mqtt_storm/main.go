package main

import (
	"flag"
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/mocker"
	"github.com/timeway/mqtt-storm/server"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
)

func init() {
	// Trace > Debug > Info > Warn > Error > Fatal > Panic
	logrus.SetLevel(logrus.InfoLevel)
	// 打印源文件
	logrus.SetReportCaller(true)
	// 指定源文件格式
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		TimestampFormat: time.DateTime,
		CallerFirst:     true,
		CustomCallerFormatter: func(frame *runtime.Frame) string {
			return fmt.Sprintf(" %s:%d", frame.File, frame.Line)
		},
	})
}

func main() {
	broker := flag.String("broker", "mqtt://127.0.0.1:1883", "URI(tcp://{ip}:{port}) of MQTT broker (required)")
	username := flag.String("username", "admin", "Username for connecting to the MQTT broker")
	password := flag.String("password", "admin", "Password for connecting to the MQTT broker")
	clientNum := flag.Uint64("c", 1, "Number of clients")
	flag.Parse()

	if *broker == "" {
		log.Printf("Invalid argument: broker is empty")
		return
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	m := mocker.NewMocker(*broker, *username, *password)
	mss := server.NewMqttStormServer(":8080", m)
	errChan, err := mss.ListenAndServe(*clientNum)
	if err != nil {
		logrus.Errorf("server error: %s", err.Error())
		return
	}

	logrus.Infof("=========>>> successfull <<<=========")
	logrus.Infof("=========>>> successfull <<<=========")
	logrus.Infof("=========>>> successfull <<<=========")

	select {
	case err = <-errChan:
		logrus.Errorf("server error: %s", err)
	case <-shutdown:
		err = mss.Shutdown()
		if err != nil {
			logrus.Errorf("shutdown error: %s", err.Error())
		}
	}

	logrus.Infof("=========>>> gameover <<<=========")
	logrus.Infof("=========>>> gameover <<<=========")
	logrus.Infof("=========>>> gameover <<<=========")
}
