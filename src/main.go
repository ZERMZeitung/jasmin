package main

import (
	"encoding/csv"
	"errors"
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

func getHTMLArticle(reqURI string) ([]byte, error) {
	//so far havent managed to exploit, but it should work in theory:
	//echo 'GET /zerm/../../../../etc/passwd HTTP/1.1
	//Host: localhost:8099
	//User-Agent: curl/7.64.1
	//Accept: */*
	//' |nc localhost 8099
	//TODO: check this
	path := "../zerm.eu/zerm" + reqURI + ".md"

	file, err := os.OpenFile(path, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	md := make([]byte, size)
	n, err := file.Read(md)
	if err != nil {
		return nil, err
	}
	if int64(n) < size {
		return nil, errors.New("Can't read the full markdown file apparently")
	}

	return markdown.ToHTML(md, nil, nil), nil
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
		} else if r.RequestURI == "/" || strings.HasPrefix(r.RequestURI, "/index") {
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
		} else if r.RequestURI == "/sitemap.xml" {
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
		} else if r.RequestURI == "/rss.xml" {
			articles, err := parseArticles()
			if err != nil {
				internalServerError(w, err)
				return
			}

			fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"utf-8\"?>")
			fmt.Fprintln(w, "<?xml-stylesheet type=\"text/css\" href=\"rss.css\" ?>")
			fmt.Fprintln(w, "<rss version=\"2.0\" xmlns:atom=\"http://www.w3.org/2005/Atom\">")
			fmt.Fprintln(w, "<channel>")
			fmt.Fprintln(w, "<title>ZERM Artikel</title>")
			fmt.Fprintln(w, "<description>Alle Artikel der Zeitung zur Erhaltung der Rechte des Menschen.</description>")
			fmt.Fprintln(w, "<language>de-de</language>")
			fmt.Fprintln(w, "<link>https://zerm.eu/rss.xml</link>")
			fmt.Fprintln(w, "<atom:link href=\"https://zerm.eu/rss.xml\" rel=\"self\" type=\"application/rss+xml\" />")

			for _, article := range articles {
				fmt.Fprintln(w, "<item>")
				fmt.Fprintf(w, "<title>%s</title>\n", article.Title)
				fmt.Fprintf(w, "<guid>https://zerm.eu/%d.html#%s</guid>\n", article.Published.Year(), article.ID)
				fmt.Fprintf(w, "<pubDate>%s</pubDate>\n", article.Published.Format("Mon, 2 Jan 2006 15:04:05 -0700"))
				fmt.Fprintln(w, "<description><![CDATA[")

				html, err := getHTMLArticle("/" + article.ID)
				if err != nil {
					fmt.Fprintln(w, err)
				} else {
					w.Write(html)
					fmt.Fprintln(w)
				}

				fmt.Fprintln(w, "]]></description>")
			}
		} else if strings.HasPrefix(r.RequestURI, "/zerm") {
			html, err := getHTMLArticle(r.RequestURI)
			if err != nil {
				internalServerError(w, err)
			} else {
				w.Write(html)
			}
		} else {
			//TODO: check this
			path := "../zerm.eu" + r.RequestURI

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
