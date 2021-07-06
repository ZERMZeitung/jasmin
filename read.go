package main

import (
	"encoding/csv"
	"os"
	"sort"
	"time"
)

func parseArticles(file string) ([]article, error) {
	f, err := os.OpenFile(file, os.O_RDONLY, 0o644)
	if err != nil {
		Err("Can't read article file:", err)
		return nil, err
	}
	defer f.Close()

	lines, err := csv.NewReader(f).ReadAll()
	if err != nil {
		Err("Can't read article csv:", err)
		return nil, err
	}

	articles := make([]article, len(lines))
	for i, line := range lines {
		pub, err := time.Parse("02.01.2006 15:04:05 MST", line[0])
		if err != nil {
			Err("Can't parse time", line[0], "of article", line[1])
			return nil, err
		}
		if time.Now().After(pub) {
			articles[i] = article{Author: line[3], URL: line[1], ID: line[4], Title: line[2], Published: pub}
		}
	}

	sort.Slice(articles, func(p, q int) bool {
		return articles[p].Published.After(articles[q].Published)
	})

	return articles, nil
}

func genShortLut(articles []article) map[string]string {
	lut := make(map[string]string, len(articles)+2)
	lut["/"] = "https://zerm.eu/"
	lut["/index"] = "https://zerm.eu/index.html"
	for _, article := range articles {
		lut["/"+article.ID] = "https://zerm.eu/zerm/" + article.URL
	}
	return lut
}
