package main

import (
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	"go.uber.org/zap"
	"net/http"
)

func sendErr(rw http.ResponseWriter, l *zap.Logger, code int, err error) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(code)
	resp := make(map[string]string)
	resp["error"] = fmt.Sprint(err)
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		l.Error("Error happened in JSON marshal.", zap.Error(err))
	}
	_, err = rw.Write(jsonResp)
	if err != nil {
		l.Error("Failed to write response.", zap.Error(err))
	}
}

func sendResult(rw http.ResponseWriter, l *zap.Logger, resp *bleve.SearchResult) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		l.Error("Error happened in JSON marshal.", zap.Error(err))
	}
	_, err = rw.Write(jsonResp)
	if err != nil {
		l.Error("Failed to write response.", zap.Error(err))
	}
}
