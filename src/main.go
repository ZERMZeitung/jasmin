package main

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
)

type article struct {
	ID        string
	Title     string
	Published time.Time
}

//caching this would improve performance and sounds like a good idea
func parseArticles() ([]article, error) {
	file, err := os.OpenFile("../zerm.eu/articles.csv", os.O_RDONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines, err := csv.NewReader(file).ReadAll()
	if err != nil {
		return nil, err
	}

	articles := make([]article, len(lines))
	for i, line := range lines {
		pub, err := time.Parse("02.01.2006 15:04:05 MST", line[0])
		if err != nil {
			return nil, err
		}
		articles[i] = article{ID: line[1], Title: line[2], Published: pub}
	}

	return articles, nil
}

func internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, err)
}

const logo = "<text class='logo1'>ZERM</text><text class='logo2'>ONLINE</text>"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "WTF are you tryin to achieve?! This is GET only.")
			return
		}

		path := r.RequestURI

		if path == "/" || strings.HasPrefix(path, "/index") {
			articles, err := parseArticles()
			if err != nil {
				internalServerError(w, err)
				return
			}

			fmt.Fprint(w, "<html><head>")
			fmt.Fprint(w, "<title>ZERM Online</title>")
			fmt.Fprint(w, "<meta charset='utf-8'/>")
			fmt.Fprint(w, "<meta name='robots' content='index,follow'/>")
			fmt.Fprint(w, "<link rel='stylesheet' type='text/css' href='style.css'>")
			fmt.Fprint(w, "</head><body>")
			fmt.Fprint(w, logo)
			fmt.Fprint(w, "<br/><br/>")
			for y := time.Now().Year(); y >= 2019; y-- {
				fmt.Fprintf(w, "<a href='%d.html'>GA %d</a> ", y, y)
			}
			fmt.Fprint(w, "<a href='rss.xml'>RSS Feed</a>")
			fmt.Fprint(w, "<ul>")

			for _, article := range articles {
				fmt.Fprint(w, "<li>")
				fmt.Fprint(w, article.Published.Format("02.01.2006"))
				fmt.Fprint(w, " &ndash; <a href='zerm/")
				fmt.Fprint(w, article.ID)
				fmt.Fprint(w, ".html'>")
				fmt.Fprint(w, article.Title)
				fmt.Fprint(w, "</a></li>")
			}

			fmt.Fprint(w, "</ul></body></html>")
		} else if path == "/sitemap.xml" {
			articles, err := parseArticles()
			if err != nil {
				internalServerError(w, err)
				return
			}

			fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			fmt.Fprintln(w, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">")
			fmt.Fprintln(w, "<url><loc>https://zerm.eu/</loc><changefreq>daily</changefreq></url>")
			y := time.Now().Year()
			fmt.Fprintf(w, "<url><loc>https://zerm.eu/%d.html</loc><changefreq>daily</changefreq></url>\n", y)
			for y--; y >= 2019; y-- {
				fmt.Fprintf(w, "<url><loc>https://zerm.eu/%d.html</loc><changefreq>monthly</changefreq></url>\n", y)
			}

			for _, article := range articles {
				fmt.Fprintf(w, "<url><loc>https://zerm.eu/zerm/%s.html</loc><changefreq>monthly</changefreq></url>\n", article.ID)
			}

			fmt.Fprintln(w, "</urlset>")
		} else if path == "/rss.xml" {
			//TODO: make rss work
		} else if strings.HasPrefix(path, "/zerm") {
			//so far havent managed to exploit, but it should work in theory:
			//echo 'GET /zerm/../../../../etc/passwd HTTP/1.1
			//Host: localhost:8099
			//User-Agent: curl/7.64.1
			//Accept: */*
			//' |nc localhost 8099
			//TODO: check this
			path = "../zerm.eu" + path + ".md"

			file, err := os.OpenFile(path, os.O_RDONLY, 0o644)
			if err != nil {
				internalServerError(w, err)
				return
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				internalServerError(w, err)
				return
			}

			md := make([]byte, stat.Size())
			//TODO: err handling
			file.Read(md)
			w.Write(markdown.ToHTML(md, nil, nil))
		} else {
			//TODO: check this
			path = "../zerm.eu" + path

			file, err := os.OpenFile(path, os.O_RDONLY, 0o644)
			if err != nil {
				internalServerError(w, err)
				return
			}
			defer file.Close()

			stat, err := file.Stat()
			if err != nil {
				internalServerError(w, err)
				return
			}

			b := make([]byte, stat.Size())
			//TODO: err handling
			file.Read(b)
			w.Write(b)
		}
	})

	http.ListenAndServe(":8099", nil)
}
