package server

import (
	"context"
	"fmt"
	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/factory"
	"github.com/timeway/mqtt-storm/mocker"
	"github.com/timeway/mqtt-storm/server/middleware"
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
	clientFactory *factory.MqttClientFactory
	srv           *http.Server
	initClientNum uint64
}

func NewMqttStormServer(addr string, mocker mocker.Mocker, clientNum uint64) *MqttStormServer {
	srv := &MqttStormServer{
		clientFactory: factory.NewMqttClientFactory(mocker),
		srv:           &http.Server{Addr: addr},
		initClientNum: clientNum,
	}

	router := mux.NewRouter()
	router.HandleFunc("/client", srv.addClient).Methods("POST")
	router.HandleFunc("/client", srv.removeClient).Methods("DELETE")

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
		mss.clientFactory.Run(mss.initClientNum)
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
		err = mss.Shutdown()
		if err != nil {
			logrus.Errorf("shutdown error: %s", err.Error())
		}
	}

	logrus.Infof("=========>>> gameover <<<=========")
	logrus.Infof("=========>>> gameover <<<=========")
	logrus.Infof("=========>>> gameover <<<=========")
}

func (mss *MqttStormServer) Shutdown() error {
	ctx, cf := context.WithTimeout(context.Background(), 15*time.Second)
	defer cf()
	err := mss.srv.Shutdown(ctx)
	mss.clientFactory.Shutdown()
	return err
}

func (mss *MqttStormServer) addClient(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	countStr := queryParams.Get("count")
	count, parseErr := strconv.ParseInt(countStr, 10, 64)
	if parseErr != nil {
		logrus.Infof("parse count error: %s", parseErr.Error())
		http.Error(w, parseErr.Error(), http.StatusBadRequest)
		return
	}

	if successCount, addClientErr := mss.clientFactory.AddClientByCount(uint64(count)); addClientErr != nil {
		responseStr := fmt.Sprintf("成功初始化客户端百分比为: %d/%d, 终止原因: %s", successCount, count, addClientErr.Error())
		http.Error(w, responseStr, http.StatusBadRequest)
		return
	}

	w.Write([]byte("ok"))
}

func (mss *MqttStormServer) removeClient(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()
	countStr := queryParams.Get("count")
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		logrus.Infof("parse count error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mss.clientFactory.RemoveClientByCount(uint64(count))

	w.Write([]byte("ok"))
}
