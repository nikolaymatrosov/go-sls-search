package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"strings"
)

const startMark = "<|startoftext|>"

type Entry struct {
	Joke string `json:"joke,omitempty"`
}

func main() {
	f, err := os.Open("data/anek.txt")
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}
	out, err := os.OpenFile("data/anek.jsonl", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	defer out.Close()
	fileScanner := bufio.NewScanner(f)

	fileScanner.Split(bufio.ScanLines)
	joke := ""
	for fileScanner.Scan() {
		l := fileScanner.Text()
		if len(l) == 0 {
			continue
		}
		if strings.HasPrefix(l, startMark) {
			if len(joke) != 0 {
				j := &Entry{
					Joke: joke,
				}
				data, err := json.Marshal(j)
				if err != nil {
					log.Fatal(err)
				}
				data = append(data, []byte("\n")...)
				_, err = out.Write(data)
				if err != nil {
					log.Fatal(err)
				}
			}
			joke = strings.TrimPrefix(l, startMark)
		} else {
			joke += l
		}
	}
}
