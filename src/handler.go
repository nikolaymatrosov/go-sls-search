package main

import (
	"errors"
	"fmt"
	"github.com/blevesearch/bleve"
	"go.uber.org/zap"
	"net/http"
	"os"
)

const indexPath = "/tmp/index"
const bucket = "sls-search"
const key = "bleve.zip"

//goland:noinspection GoUnusedExportedFunction
func SearchHandler(rw http.ResponseWriter, req *http.Request) {
	logger, _ := zap.NewProduction()

	term := req.URL.Query().Get("term")
	if len(term) == 0 {
		sendErr(rw, logger, http.StatusBadRequest, fmt.Errorf("query string parametr 'term' is missing"))
		return
	}

	err := checkCache()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err := fetchIndex(logger)
			if err != nil {
				sendErr(rw, logger, 500, err)
			}
		} else {
			sendErr(rw, logger, 500, err)
		}
	}

	index, _ := bleve.Open(indexPath)
	query := bleve.NewQueryStringQuery(term)
	searchRequest := bleve.NewSearchRequest(query)
	searchResult, err := index.Search(searchRequest)
	if err != nil {
		sendErr(rw, logger, 500, err)
	}

	sendResult(rw, logger, searchResult)
}
