package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
)

type article struct {
	Author    string
	ID        string
	Title     string
	Published time.Time
}

//TODO: caching this improves performance and sounds like a good idea generally
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
		articles[i] = article{Author: line[3], ID: line[1], Title: line[2], Published: pub}
	}

	return articles, nil
}

func quotePreprocess(raw []byte) []byte {
	cum := regexp.MustCompile("\\\\{").ReplaceAll(raw, []byte("[\\ob\\ich\\lost\\bin]("))
	return regexp.MustCompile("\\\\}").ReplaceAll(cum, []byte(")"))
}

func quotePostprocess(raw []byte) []byte {
	i := 0
	return regexp.MustCompile("\\\\ob\\\\ich\\\\lost\\\\bin").ReplaceAllFunc(raw, func(b []byte) []byte {
		i++
		return []byte(fmt.Sprintf("<sup>[%d]</sup>", i))
	})
}

func readFile(file string) ([]byte, error) {
	//prevents attacks like GET /../../../etc/passwd
	if strings.Contains(file, "..") {
		return nil, errors.New("File path contains \"..\"")
	}

	path := "../zerm.eu" + file

	f, err := os.OpenFile(path, os.O_RDONLY, 0o644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	b := make([]byte, size)
	n, err := f.Read(b)
	if err != nil {
		return nil, err
	}
	if int64(n) < size {
		return nil, errors.New("Can't read the full file apparently")
	}

	return b, nil
}

func getHTMLArticle(reqURI string) ([]byte, error) {
	md, err := readFile(reqURI + ".md")

	if err != nil {
		return nil, err
	}

	return quotePostprocess(markdown.ToHTML(quotePreprocess(md), nil, nil)), nil
}

func internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, err)
}

//TODO: logo exception for Safari because the font is broken
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
				fmt.Fprintf(w, "<a href='%d'>GA %d</a> ", y, y)
			}
			fmt.Fprint(w, "<a href='rss.xml'>RSS Feed</a>")
			fmt.Fprint(w, "<ul>")

			for _, article := range articles {
				fmt.Fprint(w, "<li>")
				fmt.Fprint(w, article.Published.Format("02.01.2006"))
				fmt.Fprint(w, " &ndash; <a href='zerm/")
				fmt.Fprint(w, article.ID)
				fmt.Fprint(w, "'>")
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

				html, err := getHTMLArticle("/zerm/" + article.ID)
				if err != nil {
					fmt.Fprintln(w, err)
				} else {
					w.Write(html)
					fmt.Fprintln(w)
				}

				fmt.Fprintln(w, "]]></description>")
			}
		} else if strings.HasPrefix(r.RequestURI, "/zerm") {
			articles, err := parseArticles()
			if err != nil {
				internalServerError(w, err)
				return
			}

			var article article
			found := false

			for _, a := range articles {
				if "/zerm/"+a.ID == r.RequestURI {
					article = a
					found = true
					break
				}
			}

			if !found {
				http.NotFound(w, r)
				return
			}

			html, err := getHTMLArticle(r.RequestURI)
			if err != nil {
				internalServerError(w, err)
				return
			}

			author, err := readFile("/authors/" + article.Author + ".html")
			if err != nil {
				internalServerError(w, err)
				return
			}

			fmt.Fprint(w, "<html><head><title>")
			fmt.Fprint(w, article.Title)
			fmt.Fprint(w, "</title>")
			fmt.Fprint(w, "<link rel='stylesheet' type='text/css' href='../style.css'>")
			fmt.Fprint(w, "</head><body><a href='/'>zurück</a><h1>")
			fmt.Fprint(w, article.Title)
			fmt.Fprint(w, "</h1>")
			w.Write(html)
			fmt.Fprint(w, "<br/><footer>von <strong>")
			w.Write(author)
			fmt.Fprint(w, "</strong></footer></body></html>")
		} else if regexp.MustCompile("^/20[0-9]{2}$").MatchString(r.RequestURI) {
			year, err := strconv.ParseUint(strings.TrimPrefix(r.RequestURI, "/"), 10, 23)
			if err != nil {
				internalServerError(w, err)
				return
			}

			articles, err := parseArticles()
			if err != nil {
				internalServerError(w, err)
				return
			}

			fmt.Fprint(w, "<html><head><title>ZERM GA ", year)
			fmt.Fprint(w, "</title><meta charset='utf-8'/>")
			fmt.Fprint(w, "<link rel='stylesheet' type='text/css' href='style.css'>")
			fmt.Fprint(w, "</head><body>")
			fmt.Fprint(w, "<text class='logo1'>ZERM</text>")
			fmt.Fprint(w, "<text class='logo2'>ONLINE</text>")
			fmt.Fprint(w, "<text class='logo1'>G</text>")
			fmt.Fprint(w, "<text class='logo2'>esamt</text>")
			fmt.Fprint(w, "<text class='logo1'>A</text>")
			fmt.Fprint(w, "<text class='logo2'>usgabe</text>")
			fmt.Fprint(w, "<text class='logo1'>", year, "</text>")

			_, e1 := readFile(fmt.Sprint("/", year, ".pdf"))
			_, e2 := readFile(fmt.Sprint("/", year, ".svg"))
			if e1 == nil && e2 == nil {
				fmt.Fprint(w, "<p><i>Die Druckversion finden Sie auch als <a href='")
				fmt.Fprint(w, year, ".pdf'>PDF</a> mit einer <a href='", year)
				fmt.Fprint(w, ".svg'>separaten Vorderseite</a>.</i></p>")
			}

			for _, article := range articles {
				if article.Published.Year() != int(year) {
					continue
				}
				fmt.Fprint(w, "<div class='entry'>")
				fmt.Fprint(w, "<h2 id='", article.ID, "'>", article.Title, "</h2>")
				fmt.Fprint(w, "<small>[<a href='#", article.ID, "'>link</a>&mdash;")
				fmt.Fprint(w, "<a href='zerm/", article.ID, ".html'>standalone</a>]")
				fmt.Fprint(w, "</small><br/>")

				//TODO: check why the speeches break
				html, err := getHTMLArticle("/zerm/" + article.ID)
				if err != nil {
					fmt.Fprintln(w, err)
				} else {
					w.Write(html)
					fmt.Fprintln(w)
				}

				fmt.Fprint(w, "<br/><footer>von <strong>", article.Author, "</strong></footer>")
				fmt.Fprint(w, "<small>", article.Published.Format("02.01.2006 15:04:05 MST"), "</small></div>")
			}

			fmt.Fprint(w, "</body></html>")
		} else {
			b, err := readFile(r.RequestURI)

			if err != nil {
				http.NotFound(w, r)
				return
			}

			w.Write(b)
		}
	})

	http.ListenAndServe(":8099", nil)
}
