package server

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/timeway/mqtt-storm/mocker"
	"github.com/timeway/mqtt-storm/server/middleware"
	"net/http"
	"strconv"
	"time"
)

type MqttStormServer struct {
	mocker *mocker.Mocker
	srv    *http.Server
}

func NewMqttStormServer(addr string, m *mocker.Mocker) *MqttStormServer {
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
	count, err := strconv.ParseInt(countStr, 10, 64)
	if err != nil {
		logrus.Infof("parse count error: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err = mss.mocker.AddClient(uint64(count)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	mss.mocker.RemoveClient(uint64(count))

	w.Write([]byte("ok"))
}
