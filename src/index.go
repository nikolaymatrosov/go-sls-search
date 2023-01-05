package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/krolaw/zipstream"
	"go.uber.org/zap"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func fetchIndex(logger *zap.Logger) error {

	var cfg, err = config.LoadDefaultConfig(context.TODO())
	if err != nil {
		logger.Error("failed to init s3 config", zap.Error(err))
		return err
	}

	client := s3.NewFromConfig(cfg)

	if err != nil {
		log.Fatal(err)
	}

	err = downloadAndUnzip(client, logger)
	if err != nil {
		logger.Error("Failed to unzip index", zap.Error(err))
		return err
	}
	return nil
}

func downloadAndUnzip(client *s3.Client, l *zap.Logger) error {
	downloader := manager.NewDownloader(client)
	pipeReader, pipeWriter := io.Pipe()

	errorChannel := make(chan error)
	wgDone := make(chan bool)
	wg := sync.WaitGroup{}

	wg.Add(2)

	go func() {
		_, err := downloader.Download(context.TODO(), PipeWriter{pipeWriter}, &s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			l.Fatal("failed to download", zap.Error(err))
		}
		wg.Done()
		err = pipeWriter.Close()
		if err != nil {
			errorChannel <- err
		}
	}()
	go func() {
		archive := zipstream.NewReader(pipeReader)

		for {
			f, err := archive.Next()
			if err != nil {
				if err != io.EOF {
					l.Fatal("failed to unzip", zap.Error(err))
					errorChannel <- err
				}
				break
			}
			filePath := filepath.Join(indexPath, f.Name)
			l.Debug("unzipping file", zap.String("filePath", filePath))

			if !strings.HasPrefix(filePath, filepath.Clean(indexPath)+string(os.PathSeparator)) {
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

			if _, err := io.Copy(dstFile, archive); err != nil {
				errorChannel <- err
			}

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

type PipeWriter struct {
	io.Writer
}

func (w PipeWriter) WriteAt(p []byte, offset int64) (n int, err error) {
	return w.Write(p)
}

func checkCache() error {
	info, err := os.Stat(indexPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("index is not dir")
	}
	return nil
}
