package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/playnet-public/libs/log"
)

type API struct {
	log *log.Logger
}

type response struct {
	Success bool             `json:"success"`
	Err     error            `json:"error"`
	Data    *json.RawMessage `json:"data"`
}

//New returns a new API
func New(log *log.Logger) API {
	return API{
		log: log,
	}
}

//Run the HTTP Server
func (a *API) Run() {
	a.log.Info("HTTP API starting.")
	r := mux.NewRouter()
	r.HandleFunc("/api/test", a.handleTest)
	r.HandleFunc("/api/example", a.handleExample)
	http.ListenAndServe(":80", r)
}

//writeResponse builds and sends a JSON message including whether the request was successful, further information about a error which might occurred and another context-specific JSON object with more data from by the API call.
func (a *API) writeResponse(writer http.ResponseWriter, success bool, err error, data *json.RawMessage) {
	resp := response{
		Success: success,
		Err:     err,
		Data:    data,
	}
	b, err := json.MarshalIndent(&resp, "", "\t")
	if err != nil {
		a.log.Error(err.Error())
	}
	writer.Write(b)
}

//handleTest is used to test if the API responds
func (a *API) handleTest(w http.ResponseWriter, r *http.Request) {
	a.writeResponse(w, true, nil, nil)
}

//handleExample is a example API function to show how to add additional json data to a response
func (a *API) handleExample(w http.ResponseWriter, r *http.Request) {
	data := struct {
		CurrentUnixTime int64
		CurrentDate     string
	}{
		CurrentUnixTime: time.Now().Unix(),
		CurrentDate:     time.Now().Format("Mon Jan 2 15:04:05 -0700 MST 2006"),
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		a.writeResponse(w, false, err, nil)
	}
	raw := json.RawMessage(jsonData)
	a.writeResponse(w, true, err, &raw)
}
