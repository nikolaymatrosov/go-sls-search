package main

import (
	"archive/zip"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/eikenb/pipeat"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const headFetchTimout = 1 * time.Second
const downloadTimeout = 5 * time.Second

func fetchIndex(ctx context.Context, logger *zap.Logger) error {
	durations := ctx.Value("durations").(map[string]int64)
	start := time.Now()
	defer func() {
		durations["fetchIndex"] = time.Now().Sub(start).Microseconds()
	}()
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

	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// We'll get keys from env variables
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		logger.Error("failed to init s3 config", zap.Error(err))
		return err
	}

	client := s3.NewFromConfig(cfg)

	if err != nil {
		log.Fatal(err)
	}

	err = downloadAndUnzip(ctx, client, logger)
	if err != nil {
		logger.Error("Failed to unzip index", zap.Error(err))
		return err
	}
	return nil
}

func downloadAndUnzip(ctx context.Context, client *s3.Client, l *zap.Logger) error {
	durations := ctx.Value("durations").(map[string]int64)
	start := time.Now()
	defer func() {
		durations["downloadAndUnzip"] = time.Now().Sub(start).Microseconds()
	}()
	downloader := manager.NewDownloader(client)
	pipeReaderAt, pipeWriterAt, err := pipeat.Pipe()
	if err != nil {
		l.Error("pipeAt error", zap.Error(err))
	}

	info, err := getObjectInfo(ctx, client)
	if err != nil {
		return err
	}

	errorChannel := make(chan error)
	wgDone := make(chan bool)
	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		downloadCtx, cancel := context.WithTimeout(ctx, downloadTimeout)
		defer cancel()
		_, err := downloader.Download(downloadCtx, pipeWriterAt, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			l.Fatal("failed to download", zap.Error(err))
		}
		wg.Done()
		err = pipeWriterAt.Close()
		if err != nil {
			errorChannel <- err
		}
	}()
	go func() {
		archive, err := zip.NewReader(pipeReaderAt, info.ContentLength)
		if err != nil {
			log.Fatal(err)
		}
		const root = "/tmp"
		for _, f := range archive.File {
			if err != nil {
				if err != io.EOF {
					l.Fatal("failed to unzip", zap.Error(err))
					errorChannel <- err
				}
				break
			}
			filePath := filepath.Join(root, f.Name)
			l.Debug(fmt.Sprintf("unzipping file %s", filePath),
				zap.String("filePath", filePath),
			)

			if !strings.HasPrefix(filePath, filepath.Clean(root)+string(os.PathSeparator)) {
				errorChannel <- fmt.Errorf("invalid file path")
			}
			if f.FileInfo().IsDir() {
				l.Debug("creating directory", zap.String("filePath", filePath))
				err := os.MkdirAll(filePath, os.ModePerm)
				if err != nil {
					errorChannel <- err
				}
				continue
			}

			if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
				errorChannel <- err
			}

			dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				errorChannel <- err
			}
			reader, err := f.Open()
			if err != nil {
				errorChannel <- err
			}
			if _, err := io.Copy(dstFile, reader); err != nil {
				errorChannel <- err
			}
			_ = reader.Close()
			err = dstFile.Close()
			if err != nil {
				errorChannel <- err
			}
		}
		wg.Done()
	}()

	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-errorChannel:
		close(errorChannel)
		return err
	}
	return nil
}

func getObjectInfo(ctx context.Context, client *s3.Client) (*s3.HeadObjectOutput, error) {
	durations := ctx.Value("durations").(map[string]int64)
	start := time.Now()
	defer func() {
		durations["getObjectInfo"] = time.Now().Sub(start).Microseconds()
	}()
	headCtx, cancel := context.WithTimeout(ctx, headFetchTimout)
	defer cancel()

	info, err := client.HeadObject(headCtx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return info, nil
}

func checkCache(ctx context.Context) error {
	durations := ctx.Value("durations").(map[string]int64)
	start := time.Now()
	defer func() {
		durations["checkCache"] = time.Now().Sub(start).Microseconds()
	}()
	info, err := os.Stat(indexPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("index is not dir")
	}
	return nil
}
