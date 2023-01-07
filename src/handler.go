package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/blevesearch/bleve"
	_ "github.com/blevesearch/bleve/analysis/analyzer/keyword"
	_ "github.com/blevesearch/bleve/analysis/lang/ru"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strconv"
	"time"
)

const indexPath = "/tmp/index"
const bucket = "sls-search"
const key = "film/index.tar.zst"

var fields = []string{
	"foreignName",
	"filmname",
	"studio",
	"crYearOfProduction",
	"director",
	"scriptAuthor",
	"composer",
	"cameraman",
	"producer",
	"duration",
	"color",
	"annotation",
	"countryOfProduction",
	"category",
	"ageLimit",
}

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
	searchRequest.Fields = fields

	yearFacet := &bleve.FacetRequest{
		Size:  20,
		Field: "crYearOfProduction",
	}
	for i := 2000; i < 2020; i++ {
		minY := float64(i)
		maxY := float64(i + 1)
		yearFacet.AddNumericRange(strconv.Itoa(i), &minY, &maxY)
	}

	searchRequest.Facets = bleve.FacetsRequest{
		"year": yearFacet,
		"country": &bleve.FacetRequest{
			Size:  5,
			Field: "countryOfProduction",
		},
	}
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
