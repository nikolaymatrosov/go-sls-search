package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

//goland:noinspection GoUnusedExportedFunction
func SpeedHandler(rw http.ResponseWriter, req *http.Request) {
	logger, _ := zap.NewProduction()
	start := time.Now()

	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				PartitionID:   "yc",
				URL:           "https://storage.yandexcloud.net",
				SigningRegion: "ru-central1",
			}, nil
		}
		return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
	})

	// We'll get keys from env variables
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		logger.Error("failed to init s3 config", zap.Error(err))
		return
	}

	client := s3.NewFromConfig(cfg)
	downloader := manager.NewDownloader(client)
	qFile := req.URL.Query().Get("file")
	file, _ := os.Create(path.Join("/tmp", qFile))
	downloadCtx, cancel := context.WithTimeout(context.Background(), downloadTimeout)
	defer cancel()
	_, err = downloader.Download(downloadCtx, file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("film/" + qFile),
	})
	if err != nil {
		logger.Fatal("failed to download", zap.Error(err))
	}
	err = file.Close()

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(200)
	resp := make(map[string]string)
	resp["time"] = strconv.Itoa(int(time.Now().Sub(start).Milliseconds()))
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		logger.Error("Error happened in JSON marshal.", zap.Error(err))
	}
	_, err = rw.Write(jsonResp)
}
