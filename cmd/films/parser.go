package main

import (
	"encoding/json"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type FilmEntry struct {
	Data struct {
		Info struct {
			CreateDate time.Time `json:"createDate"`
			UpdateDate time.Time `json:"updateDate"`
		} `json:"info"`
		General struct {
			Id                        int           `json:"id"`
			CardNumber                string        `json:"cardNumber"`
			CardDate                  time.Time     `json:"cardDate"`
			ForeignName               string        `json:"foreignName"`
			Filmname                  string        `json:"filmname"`
			Studio                    string        `json:"studio"`
			CrYearOfProduction        string        `json:"crYearOfProduction"`
			Dubbing                   string        `json:"dubbing"`
			Director                  string        `json:"director"`
			ScriptAuthor              string        `json:"scriptAuthor"`
			Composer                  string        `json:"composer"`
			Cameraman                 string        `json:"cameraman"`
			Artdirector               string        `json:"artdirector"`
			Producer                  string        `json:"producer"`
			NumberOfSeries            string        `json:"numberOfSeries"`
			NumberOfParts             string        `json:"numberOfParts"`
			Footage                   string        `json:"footage"`
			DurationMinute            string        `json:"durationMinute"`
			DurationHour              string        `json:"durationHour"`
			Color                     string        `json:"color"`
			CategoryOfRights          string        `json:"categoryOfRights"`
			AgeCategory               string        `json:"ageCategory"`
			Annotation                string        `json:"annotation"`
			Remark                    string        `json:"remark"`
			PsSdateTo                 string        `json:"psSdateTo"`
			ViewMovie                 string        `json:"viewMovie"`
			CountryOfProduction       string        `json:"countryOfProduction"`
			Category                  string        `json:"category"`
			CadrFormat                string        `json:"cadrFormat"`
			StartDateRent             time.Time     `json:"startDateRent"`
			Owner                     string        `json:"owner"`
			CrRentalRightsTransferred []interface{} `json:"crRentalRightsTransferred"`
			Deleted                   bool          `json:"deleted"`
			DoNotShowOnSite           bool          `json:"doNotShowOnSite"`
			AgeLimit                  string        `json:"ageLimit"`
		} `json:"general"`
	} `json:"data"`
}
type Film struct {
	DocType             string `json:"_type"`
	ForeignName         string `json:"foreignName"`
	Filmname            string `json:"filmname"`
	Studio              string `json:"studio"`
	CrYearOfProduction  int    `json:"crYearOfProduction"`
	Director            string `json:"director"`
	ScriptAuthor        string `json:"scriptAuthor"`
	Composer            string `json:"composer"`
	Cameraman           string `json:"cameraman"`
	Producer            string `json:"producer"`
	Duration            int    `json:"duration"`
	Color               string `json:"color"`
	Annotation          string `json:"annotation"`
	CountryOfProduction string `json:"countryOfProduction"`
	Category            string `json:"category"`
	AgeLimit            int    `json:"ageLimit"`
}

func (e FilmEntry) ToFilm() Film {
	ageLimit, _ := strconv.Atoi(e.Data.General.AgeLimit)
	mins, _ := strconv.Atoi(e.Data.General.DurationMinute)
	hours, _ := strconv.Atoi(e.Data.General.DurationHour)
	duration := hours*60 + mins
	year, _ := strconv.Atoi(e.Data.General.CrYearOfProduction)
	return Film{
		DocType:             "film",
		ForeignName:         e.Data.General.ForeignName,
		Filmname:            e.Data.General.Filmname,
		Studio:              e.Data.General.Studio,
		CrYearOfProduction:  year,
		Director:            e.Data.General.Director,
		ScriptAuthor:        e.Data.General.ScriptAuthor,
		Composer:            e.Data.General.Composer,
		Producer:            e.Data.General.Producer,
		Duration:            duration,
		Color:               e.Data.General.Color,
		Annotation:          e.Data.General.Annotation,
		CountryOfProduction: strings.ReplaceAll(e.Data.General.CountryOfProduction, "-", " "),
		Category:            e.Data.General.Category,
		AgeLimit:            ageLimit,
	}
}

func main() {
	const dir = "data/films/film_approvals.json/"
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	out, _ := os.Create(path.Join(dir, "result.jsonl"))
	defer out.Close()

	for _, file := range files {
		entries := []FilmEntry{}
		contents, _ := os.ReadFile(path.Join(dir, file.Name()))

		err := json.Unmarshal(contents, &entries)
		if err != nil {
			log.Fatal(err)
		}
		for _, e := range entries {
			data, err := json.Marshal(e.ToFilm())
			if err != nil {
				log.Fatal(err)
			}
			data = append(data, []byte("\n")...)
			_, err = out.Write(data)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
