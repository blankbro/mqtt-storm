package response

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
)

const (
	Success int32 = 200
	Error   int32 = 500
)

type responseData struct {
	Code int32       `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func SuccessResponse(w http.ResponseWriter, data interface{}) {
	response := responseData{
		Code: Success,
		Msg:  "success",
		Data: data,
	}

	writeResponse(w, response)
}

func ErrorResponse(w http.ResponseWriter, message string) {
	logrus.Warn(message)

	response := responseData{
		Code: Error,
		Msg:  message,
	}

	writeResponse(w, response)
}

func writeResponse(w http.ResponseWriter, response responseData) {
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(
			w,
			fmt.Sprintf("Internal Server Error: %s", err.Error()),
			http.StatusInternalServerError,
		)
	}
}
