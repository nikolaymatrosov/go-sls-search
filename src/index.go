package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/eikenb/pipeat"
	"github.com/mholt/archiver/v4"
	"go.uber.org/zap"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"sync"
	"time"
)

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
		logger.Error("Failed to download and unzip index", zap.Error(err))
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
		const root = "/tmp"
		format, input, err := archiver.Identify(key, pipeReaderAt)
		if err != nil {
			l.Error("unsupported archive type for file",
				zap.String("source", key),
				zap.Error(err))
			errorChannel <- err
		}

		if ex, ok := format.(archiver.Extractor); ok {
			err := ex.Extract(context.Background(), input, nil, func(ctx context.Context, f archiver.File) error {
				if f.IsDir() {
					return nil
				}

				outputPath := path.Join(root, f.NameInArchive)

				// create symlinks
				if f.LinkTarget != "" {
					err := os.Symlink(f.LinkTarget, outputPath)
					if err != nil {
						return err
					}
					return nil
				}

				reader, err := f.Open()
				if err != nil {
					return err
				}
				defer reader.Close()

				writer, err := safeCreateFile(outputPath, f.Mode())
				if err != nil {
					return fmt.Errorf("failed to create %v: %v", outputPath, err)
				}
				defer writer.Close()

				if _, err := io.Copy(writer, reader); err != nil {
					return fmt.Errorf("failed to write %v: %v", outputPath, err)
				}

				return nil
			})
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

func safeCreateFile(filePath string, mode fs.FileMode) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %v: %v", path.Dir(filePath), err)
	}

	return os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
}
