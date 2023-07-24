package server

import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/factory"
	"github.com/timeway/mqtt-storm/server/middleware"
	"net/http"
	"strconv"
	"time"
)

type MqttStormServer struct {
	mocker *factory.MqttClientFactory
	srv    *http.Server
}

func NewMqttStormServer(addr string, m *factory.MqttClientFactory) *MqttStormServer {
	srv := &MqttStormServer{
		mocker: m,
		srv:    &http.Server{Addr: addr},
	}

	router := mux.NewRouter()
	router.HandleFunc("/client", srv.addClient).Methods("POST")
	router.HandleFunc("/client", srv.removeClient).Methods("DELETE")

	srv.srv.Handler = middleware.Logging(router)

	return srv
}

func (mss *MqttStormServer) ListenAndServe(clientNum uint64) (<-chan error, error) {
	errChan := make(chan error)
	go func() {
		err := mss.srv.ListenAndServe()
		errChan <- err
	}()

	var err error
	select {
	case err = <-errChan:
		return nil, err
	case <-time.After(1 * time.Second):
		mss.mocker.Run(clientNum)
		return errChan, nil
	}
}

func (mss *MqttStormServer) Shutdown() error {
	ctx, cf := context.WithTimeout(context.Background(), 15*time.Second)
	defer cf()
	err := mss.srv.Shutdown(ctx)
	mss.mocker.Shutdown()
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

	if successCount, addClientErr := mss.mocker.AddClientByCount(uint64(count)); addClientErr != nil {
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

	mss.mocker.RemoveClientByCount(uint64(count))

	w.Write([]byte("ok"))
}
