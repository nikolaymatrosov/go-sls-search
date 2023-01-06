package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/blevesearch/bleve"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"
)

const indexPath = "/tmp/index"
const bucket = "sls-search"
const key = "bleve.zip"

//goland:noinspection GoUnusedExportedFunction
func SearchHandler(rw http.ResponseWriter, req *http.Request) {
	logger, _ := zap.NewProduction()
	ctx := context.WithValue(context.Background(), "durations", map[string]int64{})

	term := req.URL.Query().Get("term")
	if len(term) == 0 {
		sendErr(ctx, rw, logger, http.StatusBadRequest, fmt.Errorf("query string parametr 'term' is missing"))
		return
	}

	err := checkCache(ctx)
	if err != nil {
		logger.Error("check result", zap.Error(err))
		if errors.Is(err, os.ErrNotExist) {
			err := fetchIndex(ctx, logger)
			if err != nil {
				sendErr(ctx, rw, logger, 500, err)
			}
		} else {
			sendErr(ctx, rw, logger, 500, err)
		}
	} else {
		logger.Info("cache hit")
	}

	durations := ctx.Value("durations").(map[string]int64)
	start := time.Now()
	index, err := bleve.Open(indexPath)
	if err != nil {
		logger.Error("error open index", zap.Error(err))
	}
	durations["openIndex"] = time.Now().Sub(start).Microseconds()

	start = time.Now()
	query := bleve.NewQueryStringQuery(term)
	searchRequest := bleve.NewSearchRequest(query)
	searchRequest.Fields = []string{"joke"}
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		sendErr(ctx, rw, logger, 500, err)
	}
	durations["queryIndex"] = time.Now().Sub(start).Microseconds()
	start = time.Now()
	err = index.Close()
	if err != nil {
		logger.Error("error closing index", zap.Error(err))
	}
	durations["closeIndex"] = time.Now().Sub(start).Microseconds()
	sendResult(ctx, rw, logger, searchResult)
}
