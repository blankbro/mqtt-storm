package server

import (
	"context"
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/mocker"
	"github.com/timeway/mqtt-storm/server/middleware"
	"github.com/timeway/mqtt-storm/server/response"
	"github.com/timeway/mqtt-storm/storm"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
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

type MqttStormServer struct {
	mqttStorm     *storm.MqttStorm
	srv           *http.Server
	initClientNum uint64
}

func NewMqttStormServer(addr string, mocker mocker.Mocker, clientNum uint64) *MqttStormServer {
	srv := &MqttStormServer{
		mqttStorm:     storm.NewMqttStorm(mocker),
		srv:           &http.Server{Addr: addr},
		initClientNum: clientNum,
	}

	router := mux.NewRouter()
	router.HandleFunc("/client", srv.addClient).Methods("POST")
	router.HandleFunc("/client", srv.removeClient).Methods("DELETE")
	router.HandleFunc("/sub", srv.subStorm).Methods("POST")
	router.HandleFunc("/pub", srv.pubStorm).Methods("POST")

	srv.srv.Handler = middleware.Logging(router)

	return srv
}

func (mss *MqttStormServer) ListenAndServe() {
	errChan := make(chan error)
	go func() {
		err := mss.srv.ListenAndServe()
		errChan <- err
	}()

	var err error
	select {
	case err = <-errChan:
		logrus.Errorf("server error: %s", err.Error())
		return
	case <-time.After(1 * time.Second):
		mss.mqttStorm.Run(mss.initClientNum)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	logrus.Infof("=========>>> successfull <<<=========")
	logrus.Infof("=========>>> successfull <<<=========")
	logrus.Infof("=========>>> successfull <<<=========")

	select {
	case err = <-errChan:
		logrus.Errorf("server error: %s", err.Error())
	case <-shutdown:
		err = mss.shutdown()
		if err != nil {
			logrus.Errorf("shutdown error: %s", err.Error())
		}
	}

	logrus.Infof("=========>>> gameover <<<=========")
	logrus.Infof("=========>>> gameover <<<=========")
	logrus.Infof("=========>>> gameover <<<=========")
}

func (mss *MqttStormServer) shutdown() error {
	ctx, cf := context.WithTimeout(context.Background(), 15*time.Second)
	defer cf()
	err := mss.srv.Shutdown(ctx)
	mss.mqttStorm.Shutdown()
	return err
}

func (mss *MqttStormServer) addClient(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	countStr := queryParams.Get("count")
	count, parseErr := strconv.ParseInt(countStr, 10, 64)
	if parseErr != nil {
		errInfo := fmt.Sprintf("parse count error: %s", parseErr.Error())
		response.ErrorResponse(w, errInfo)
		return
	}

	if successCount, addClientErr := mss.mqttStorm.AddClientByCount(uint64(count)); addClientErr != nil {
		errInfo := fmt.Sprintf("成功初始化客户端百分比为: %d/%d, 终止原因: %s", successCount, count, addClientErr.Error())
		response.ErrorResponse(w, errInfo)
		return
	}

	response.SuccessResponse(w, nil)
}

func (mss *MqttStormServer) removeClient(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	countStr := queryParams.Get("count")
	count, parseErr := strconv.ParseInt(countStr, 10, 64)
	if parseErr != nil {
		errInfo := fmt.Sprintf("parse count error: %s", parseErr.Error())
		response.ErrorResponse(w, errInfo)
		return
	}

	mss.mqttStorm.RemoveClientByCount(uint64(count))

	response.SuccessResponse(w, nil)
}

func (mss *MqttStormServer) subStorm(w http.ResponseWriter, _ *http.Request) {
	successCount, totalCount, err := mss.mqttStorm.SubStorm()
	if err != nil {
		errInfo := fmt.Sprintf("成功订阅的占比为: %d/%d, 终止原因: %s", successCount, totalCount, err.Error())
		response.ErrorResponse(w, errInfo)
		return
	}

	response.SuccessResponse(w, nil)
}

func (mss *MqttStormServer) pubStorm(w http.ResponseWriter, r *http.Request) {
	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		errInfo := fmt.Sprintf("read request body error: %s", readErr.Error())
		response.ErrorResponse(w, errInfo)
	}

	pubErr := mss.mqttStorm.PubStorm(body)
	if pubErr != nil {
		response.ErrorResponse(w, pubErr.Error())
	}

	response.SuccessResponse(w, nil)
}
