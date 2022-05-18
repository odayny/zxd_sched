package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func get_document(url string) goquery.Selection {
	log.Printf("Requesting %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Unable to retrieve schedule page web: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatalf("Unable to parse a doc: %v", err)
	}
	// schedule table is in .lineup
	return *doc.Find(".lineup").First()
}

func getStagesList(doc goquery.Selection) []string {
	stageSection := doc.Find(".stages__menu").First().Children()
	stageList := make([]string, stageSection.Size())

	stageSection.Each(func(i int, s *goquery.Selection) {
		stageList[i] = s.Find(".stage__title").Text()
	})
	// the first stage is empty. it's ok, I'll utilize it later
	return stageList
}

func getShowList(docs [4]goquery.Selection, stages []string, startDay int, month int, year int, tz string) []Show {
	result := make([]Show, 0)
	for i, doc := range docs {
		// this year graspop begins on 16th of June
		day := i + startDay
		doc.Find(".schedule").Find(".stage").Each(func(j int, s *goquery.Selection) {
			// first column is just hours. nothing interesting
			if j == 0 {
				return
			}
			// get the stage name
			stage := stages[j]
			s.Find(".schedule__band").Each(func(_ int, s *goquery.Selection) {
				anchor := s.Find("a")
				name := anchor.Children().First().Text()
				times := anchor.Children().Last().Text()
				url, _ := anchor.Attr("href")
				startDate64, endDate64 := convertDates(year, month, day, times, tz)
				result = append(result, Show{stage,
					strconv.Itoa(j),
					name, startDate64, endDate64, url})
			})
		})
	}
	return result
}

func convertDates(year int, month int, day int, times string, tzname string) (int64, int64) {
	// ex. 13:00 - 13:40
	timeSplit := strings.Split(times, "-")
	startHour, _ := strconv.Atoi(strings.Split(strings.Trim(timeSplit[0], " "), ":")[0])
	startMinute, _ := strconv.Atoi(strings.Split(strings.Trim(timeSplit[0], " "), ":")[1])
	endHour, _ := strconv.Atoi(strings.Split(strings.Trim(timeSplit[1], " "), ":")[0])
	endMinute, _ := strconv.Atoi(strings.Split(strings.Trim(timeSplit[1], " "), ":")[1])
	tz, _ := time.LoadLocation(tzname)
	startDate := time.Date(year, time.Month(month), day, startHour, startMinute, 0, 0, tz)
	// in case show starts (ends) during next day
	if startHour < 11 {
		startDate = startDate.AddDate(0, 0, 1)
	}
	endDate := time.Date(year, time.Month(month), day, endHour, endMinute, 0, 0, tz)
	if endHour < 11 {
		endDate = endDate.AddDate(0, 0, 1)
	}
	if startDate.Unix() > endDate.Unix() {
		log.Fatalf("Wrong dates: %v, %v", startDate, endDate)
	}
	return startDate.Unix(), endDate.Unix()
}
