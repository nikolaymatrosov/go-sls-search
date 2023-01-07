package main

import (
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"github.com/mholt/archiver/v4"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

const root = "/tmp"
const index = "data/films/index"

func main() {

	res := map[string][]string{}

	var archives = map[string]archiver.CompressedArchive{
		"data/films/index.tar.gz": {
			Compression: archiver.Gz{
				CompressionLevel: gzip.BestCompression,
				Multithreaded:    true,
			},
			Archival: archiver.Tar{},
		},
		"data/films/index.tar.br": {
			Compression: archiver.Brotli{Quality: 10}, // q:11 will be archiving too long
			Archival:    archiver.Tar{},
		},
		"data/films/index.tar.zst": {
			Compression: archiver.Zstd{
				EncoderOptions: []zstd.EOption{zstd.WithEncoderLevel(zstd.SpeedBestCompression)},
			},
			Archival: archiver.Tar{},
		},
	}
	for f := range archives {
		res[f] = []string{}
	}
	res["data/films/index.zip"] = []string{}
	for i := 0; i < 10; i++ {
		for filename, format := range archives {
			archive(filename, format, res)
		}

		archiveZip("data/films/index.zip", res)
	}
	printRes(res)

	files := []string{
		"data/films/index.zip",
		//"data/films/index.7z",
		"data/films/index.tar.gz",
		"data/films/index.tar.br",
		"data/films/index.tar.zst",
	}
	res = map[string][]string{}
	for _, f := range files {
		res[f] = []string{}
	}

	for i := 0; i < 10; i++ {
		for _, filename := range files {
			extract(filename, res)
		}
	}
	printRes(res)
}

func printRes(res map[string][]string) {
	for fn, data := range res {
		fmt.Printf("%s\t%s\n", fn, strings.Join(data, "\t"))
	}
}

func archive(filename string, format archiver.CompressedArchive, res map[string][]string) {
	store := path.Join(index, "store")
	files, err := os.ReadDir(store)
	if err != nil {
		log.Fatal(err)
	}
	m := map[string]string{
		path.Join(index, "index_meta.json"): path.Join("index", "index_meta.json"),
	}

	for _, file := range files {
		m[path.Join(store, file.Name())] = path.Join("index", "store", file.Name())
	}

	af, err := archiver.FilesFromDisk(nil, m)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	start := time.Now()

	defer func() {
		delta := time.Now().Sub(start)
		fmt.Printf("%s %s\n", filename, delta)
		res[filename] = append(res[filename], strconv.Itoa(int(delta.Milliseconds())))
	}()
	// create the archive
	err = format.Archive(context.Background(), out, af)
	if err != nil {
		log.Fatal(err)
	}
}

func archiveZip(filename string, res map[string][]string) {
	store := path.Join(index, "store")
	files, err := os.ReadDir(store)
	if err != nil {
		log.Fatal(err)
	}
	m := map[string]string{
		path.Join(index, "index_meta.json"): path.Join("index", "index_meta.json"),
	}

	for _, file := range files {
		m[path.Join(store, file.Name())] = path.Join("index", "store", file.Name())
	}

	af, err := archiver.FilesFromDisk(nil, m)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	start := time.Now()

	defer func() {
		delta := time.Now().Sub(start)
		fmt.Printf("%s %s\n", filename, delta)
		res[filename] = append(res[filename], strconv.Itoa(int(delta.Milliseconds())))
	}()
	// create the archive
	format := archiver.Zip{
		Compression:          zip.Deflate,
		SelectiveCompression: true,
	}
	err = format.Archive(context.Background(), out, af)
	if err != nil {
		log.Fatal(err)
	}
}

func extract(filename string, res map[string][]string) {
	inp, err := os.Open(filename)
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}
	format, input, err := archiver.Identify(filename, inp)
	if err != nil {
		log.Fatalf("unsupported archive type for file: %s", err)
		return
	}
	start := time.Now()

	defer func() {
		delta := time.Now().Sub(start)
		fmt.Printf("%s %s\n", filename, delta)
		res[filename] = append(res[filename], strconv.Itoa(int(delta.Milliseconds())))
	}()

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
			log.Fatalf("failed extract: %s", err)
			return
		}
	}
}

func safeCreateFile(filePath string, mode fs.FileMode) (*os.File, error) {
	if err := os.MkdirAll(path.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %v: %v", path.Dir(filePath), err)
	}

	return os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
}
