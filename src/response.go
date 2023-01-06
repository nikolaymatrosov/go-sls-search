package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/blevesearch/bleve"
	"go.uber.org/zap"
	"net/http"
)

func sendErr(_ context.Context, rw http.ResponseWriter, l *zap.Logger, code int, err error) {
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

type SearchResultWithTimings struct {
	*bleve.SearchResult
	Durations map[string]int64 `json:"durations"`
}

func sendResult(ctx context.Context, rw http.ResponseWriter, l *zap.Logger, resp *bleve.SearchResult) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	durations := ctx.Value("durations").(map[string]int64)
	withTimings := SearchResultWithTimings{
		resp,
		durations,
	}

	jsonResp, err := json.Marshal(withTimings)
	if err != nil {
		l.Error("Error happened in JSON marshal.", zap.Error(err))
	}
	_, err = rw.Write(jsonResp)
	if err != nil {
		l.Error("Failed to write response.", zap.Error(err))
	}
}
