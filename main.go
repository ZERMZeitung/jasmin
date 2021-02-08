package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
)

//TODO:
// - logging
// - logo exception for Safari because the font is broken
// - cache md or maybe even html articles

type article struct {
	Author    string
	URL       string
	Title     string
	ID        string
	Published time.Time
}

var allArticles []article
var shortLut map[string]string
var lastUpdate = time.Unix(0, 0)

func parseArticles() ([]article, error) {
	lines, err := readCsv("/articles.csv")
	if err != nil {
		return nil, err
	}

	articles := make([]article, len(lines))
	for i, line := range lines {
		pub, err := time.Parse("02.01.2006 15:04:05 MST", line[0])
		if err != nil {
			return nil, err
		}
		articles[i] = article{Author: line[3], URL: line[1], ID: line[4], Title: line[2], Published: pub}
	}

	return articles, nil
}

func genShortLut() (map[string]string, error) {
	articles := allArticles
	lut := make(map[string]string, len(articles)+2)
	lut["/"] = "https://zerm.eu/"
	lut["/index"] = "https://zerm.eu/index.html"
	for _, article := range articles {
		lut["/"+article.ID] = "https://zerm.eu/zerm/" + article.URL
	}
	return lut, nil
}

func update() error {
	log.Println("[update()] updating...")
	articles, err := parseArticles()
	if err != nil {
		log.Println("[update()] articles: ", err)
		return err
	}

	allArticles = articles

	lut, err := genShortLut()
	if err != nil {
		log.Println("[update()] short lut: ", err)
		return err
	}

	shortLut = lut
	lastUpdate = time.Now()
	log.Println("[update()] done.")
	return nil
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

func openFile(file string) (*os.File, error) {
	//prevents attacks like GET /../../../etc/passwd
	if strings.Contains(file, "..") {
		return nil, errors.New("File path contains \"..\"")
	}

	return os.OpenFile("../zerm.eu"+file, os.O_RDONLY, 0o644)
}

func readCsv(file string) ([][]string, error) {
	f, err := openFile(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return csv.NewReader(f).ReadAll()
}

func readFile(file string) ([]byte, error) {
	f, err := openFile(file)
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

const logo = "<text class='logo1'>ZERM</text><text class='logo2'>ONLINE</text>"

func main() {
	err := update()
	if err != nil {
		log.Fatalln(err)
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Got an %s request from %s (%s): %s (%s)",
			r.Proto, r.RemoteAddr, r.UserAgent(), r.URL.Path, r.Host)
		if lastUpdate.Add(1000_000000).Before(time.Now()) {
			update()
		}
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "WTF are you tryin to achieve?! This is GET only.")
		} else if strings.Contains(r.Host, "zerm.link") {
			url, ok := shortLut[r.URL.Path]
			if !ok {
				log.Printf("%s not found.", r.URL.Path)
				w.WriteHeader(404)
				fmt.Fprintf(w, "Couldn't find what you were looking for.")
				return
			}
			log.Printf("Redirecting: %s", url)
			http.Redirect(w, r, url, 307)
		} else if r.RequestURI == "/" || strings.HasPrefix(r.RequestURI, "/index") {
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

			for _, article := range allArticles {
				fmt.Fprint(w, "<li>")
				fmt.Fprint(w, article.Published.Format("02.01.2006"))
				fmt.Fprint(w, " &ndash; <a href='zerm/")
				fmt.Fprint(w, article.URL)
				fmt.Fprint(w, "'>")
				fmt.Fprint(w, article.Title)
				fmt.Fprint(w, "</a></li>")
			}

			fmt.Fprint(w, "</ul></body></html>")
		} else if r.RequestURI == "/sitemap.xml" {
			fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			fmt.Fprintln(w, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">")
			fmt.Fprintln(w, "<url><loc>https://zerm.eu/</loc><changefreq>daily</changefreq></url>")
			y := time.Now().Year()
			fmt.Fprintf(w, "<url><loc>https://zerm.eu/%d.html</loc><changefreq>daily</changefreq></url>\n", y)
			for y--; y >= 2019; y-- {
				fmt.Fprintf(w, "<url><loc>https://zerm.eu/%d.html</loc><changefreq>monthly</changefreq></url>\n", y)
			}

			for _, article := range allArticles {
				fmt.Fprintf(w, "<url><loc>https://zerm.eu/zerm/%s.html</loc><changefreq>monthly</changefreq></url>\n", article.URL)
			}

			fmt.Fprintln(w, "</urlset>")
		} else if r.RequestURI == "/rss.xml" {
			fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"utf-8\"?>")
			fmt.Fprintln(w, "<?xml-stylesheet type=\"text/css\" href=\"rss.css\" ?>")
			fmt.Fprintln(w, "<rss version=\"2.0\" xmlns:atom=\"http://www.w3.org/2005/Atom\">")
			fmt.Fprintln(w, "<channel>")
			fmt.Fprintln(w, "<title>ZERM Artikel</title>")
			fmt.Fprintln(w, "<description>Alle Artikel der Zeitung zur Erhaltung der Rechte des Menschen.</description>")
			fmt.Fprintln(w, "<language>de-de</language>")
			fmt.Fprintln(w, "<link>https://zerm.eu/rss.xml</link>")
			fmt.Fprintln(w, "<atom:link href=\"https://zerm.eu/rss.xml\" rel=\"self\" type=\"application/rss+xml\" />")

			for _, article := range allArticles {
				fmt.Fprintln(w, "<item>")
				fmt.Fprintf(w, "<title>%s</title>\n", article.Title)
				fmt.Fprintf(w, "<guid>https://zerm.eu/%d.html#%s</guid>\n", article.Published.Year(), article.URL)
				fmt.Fprintf(w, "<pubDate>%s</pubDate>\n", article.Published.Format("Mon, 2 Jan 2006 15:04:05 -0700"))
				fmt.Fprintln(w, "<description><![CDATA[")

				html, err := getHTMLArticle("/zerm/" + article.URL)
				if err != nil {
					fmt.Fprintln(w, err)
				} else {
					w.Write(html)
					fmt.Fprintln(w)
				}

				fmt.Fprintln(w, "]]></description>")
			}
		} else if strings.HasPrefix(r.RequestURI, "/zerm") {
			var article article
			found := false

			for _, a := range allArticles {
				if "/zerm/"+a.URL == r.RequestURI {
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

			for _, article := range allArticles {
				if article.Published.Year() != int(year) {
					continue
				}
				fmt.Fprint(w, "<div class='entry'>")
				fmt.Fprint(w, "<h2 id='", article.URL, "'>", article.Title, "</h2>")
				fmt.Fprint(w, "<small>[<a href='#", article.URL, "'>link</a>&mdash;")
				fmt.Fprint(w, "<a href='zerm/", article.URL, ".html'>standalone</a>]")
				fmt.Fprint(w, "</small><br/>")

				html, err := getHTMLArticle("/zerm/" + article.URL)
				if err != nil {
					fmt.Fprintln(w, err)
				} else {
					w.Write(html)
					fmt.Fprintln(w)
				}

				author, err := readFile("/authors/" + article.Author + ".html")
				if err != nil {
					internalServerError(w, err)
				} else {
					fmt.Fprint(w, "<small>von <strong>")
					w.Write(author)
					fmt.Fprint(w, "</strong></small><br/>")
				}

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