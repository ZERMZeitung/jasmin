package main

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gomarkdown/markdown"
)

//TODO:
// - sort articles by datetime
// - logo exception for Safari because the font is broken
// - put a lot of the static html in separate files

type article struct {
	Author    string
	URL       string
	Title     string
	ID        string
	Published time.Time
}

var allArticles []article
var shortLut map[string]string
var htmlCache map[string][]byte
var lastUpdate = time.Unix(0, 0)
var rootDir = "/var/www/zerm.eu"

func update() error {
	articles, err := parseArticles(rootDir + "/articles.csv")
	if err != nil {
		Err("Can't update articles: ", err)
		return err
	}

	shortLut = genShortLut(articles)
	allArticles = articles
	htmlCache = make(map[string][]byte)

	lastUpdate = time.Now()
	Info("Flushed the cache.")
	return nil
}

func articlePreprocess(raw []byte) []byte {
	cum := regexp.MustCompile(`\\,`).ReplaceAll(raw, []byte("„"))
	cum = regexp.MustCompile(`\\'`).ReplaceAll(cum, []byte("“"))
	cum = regexp.MustCompile(`\\{`).ReplaceAll(cum, []byte(`[\ob\ich\lost\bin](`))
	return regexp.MustCompile(`\\}`).ReplaceAll(cum, []byte(")"))
}

func articlePostprocess(raw []byte) []byte {
	i := 0
	return regexp.MustCompile(`\\ob\\ich\\lost\\bin`).ReplaceAllFunc(raw, func(b []byte) []byte {
		i++
		return []byte(fmt.Sprintf("<sup>[%d]</sup>", i))
	})
}

func readFile(file string) ([]byte, error) {
	//prevents attacks like GET /../../../etc/passwd
	if strings.Contains(file, "..") {
		Warn("Refusing to read file containing \"..\":", file)
		return nil, errors.New("file path contains \"..\"")
	}

	f, err := os.OpenFile(rootDir+file, os.O_RDONLY, 0o644)
	if err != nil {
		Warn("Can't open file", file)
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		Err("Stat failed for \"", file, "\": ", err)
		return nil, err
	}
	size := stat.Size()

	b := make([]byte, size)
	n, err := f.Read(b)
	if err != nil {
		Err("Can't read the file \"", file, "\"")
		return nil, err
	}
	if int64(n) < size {
		Err("Can't read the file \"", file, "\" fully, wtf?")
		return nil, errors.New("can't read the full file apparently")
	}

	return b, nil
}

func getHTMLArticle(reqURI string) ([]byte, error) {
	reqURI = strings.TrimSuffix(reqURI, ".html")
	if !strings.HasSuffix(reqURI, ".md") {
		reqURI += ".md"
	}

	html, ok := htmlCache[reqURI]
	if ok {
		Info("The HTML cache actually helped!")
		return html, nil
	}

	md, err := readFile(reqURI)
	if err != nil {
		Warn("Can't read HTML article ", reqURI, ": ", err)
		return nil, err
	}

	html = articlePostprocess(markdown.ToHTML(articlePreprocess(md), nil, nil))
	htmlCache[reqURI] = html
	return html, nil
}

func writeHeader(w http.ResponseWriter, r *http.Request, code int, info string, contentType string) {
	w.WriteHeader(code)
	w.Header().Add("Content-Type", contentType+"; charset=utf-8")
	responses.WithLabelValues(fmt.Sprint(code), info, contentType, r.RequestURI).Inc()
}

func internalServerError(w http.ResponseWriter, r *http.Request, err error) {
	writeHeader(w, r, 500, fmt.Sprint(err), "text/plain")
	fmt.Fprintln(w, err)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	Warn(r.RequestURI, "not found.")
	http.NotFound(w, r)
	responses.WithLabelValues("404", "", "", r.RequestURI).Inc()
}

func redirect(w http.ResponseWriter, r *http.Request, url string, code int) {
	Info("Redirecting:", url)
	http.Redirect(w, r, url, code)
	responses.WithLabelValues(fmt.Sprint(code), url, "", r.RequestURI).Inc()
}

const logo = "<text class='logo1'>ZERM</text> <text class='logo2'>ONLINE</text>"

func main() {
	if len(os.Args) > 1 {
		rootDir = os.Args[1]
	}

	err := update()
	if err != nil {
		Fatal(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Info(fmt.Sprintf("Got an %s request from %s (%s): %s (%s)",
			r.Proto, r.RemoteAddr, r.UserAgent(), r.URL.Path, r.Host))
		userAgents.WithLabelValues(r.UserAgent()).Inc()
		requests.WithLabelValues(r.Method, r.RequestURI).Inc()
		if lastUpdate.Add(60 * time.Second).Before(time.Now()) {
			update()
		}
		if r.Method != "GET" {
			writeHeader(w, r, 400, "", "text/plain")
			fmt.Fprintln(w, "WTF are you tryin to achieve?! This is GET only.")
		} else if strings.Contains(r.Host, "link") || strings.HasPrefix(r.RequestURI, "/apache_slaughters_kittens") {
			url, ok := shortLut[strings.TrimPrefix(r.RequestURI, "/apache_slaughters_kittens")]
			if !ok {
				notFound(w, r)
				return
			}
			redirect(w, r, url, 307)
		} else if r.RequestURI == "/" || strings.HasPrefix(r.RequestURI, "/index") {
			writeHeader(w, r, 200, "", "text/html")

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

			articles := allArticles

			for _, article := range articles {
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
			writeHeader(w, r, 200, "", "text/xml")

			fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
			fmt.Fprintln(w, "<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\">")
			fmt.Fprintln(w, "<url><loc>https://zerm.eu/</loc><changefreq>daily</changefreq></url>")
			y := time.Now().Year()
			fmt.Fprintf(w, "<url><loc>https://zerm.eu/%d.html</loc><changefreq>daily</changefreq></url>\n", y)
			for y--; y >= 2019; y-- {
				fmt.Fprintf(w, "<url><loc>https://zerm.eu/%d.html</loc><changefreq>monthly</changefreq></url>\n", y)
			}

			articles := allArticles

			for _, article := range articles {
				fmt.Fprintf(w, "<url><loc>https://zerm.eu/zerm/%s.html</loc><changefreq>monthly</changefreq></url>\n", article.URL)
			}

			fmt.Fprintln(w, "</urlset>")
		} else if r.RequestURI == "/rss.xml" {
			writeHeader(w, r, 200, "", "application/rss+xml")

			fmt.Fprintln(w, "<?xml version=\"1.0\" encoding=\"utf-8\"?>")
			fmt.Fprintln(w, "<?xml-stylesheet type=\"text/css\" href=\"rss.css\" ?>")
			fmt.Fprintln(w, "<rss version=\"2.0\" xmlns:atom=\"http://www.w3.org/2005/Atom\">")
			fmt.Fprintln(w, "<channel>")
			fmt.Fprintln(w, "<title>ZERM Artikel</title>")
			fmt.Fprintln(w, "<description>Alle Artikel der Zeitung zur Erhaltung der Rechte des Menschen.</description>")
			fmt.Fprintln(w, "<language>de-de</language>")
			fmt.Fprintln(w, "<link>https://zerm.eu/rss.xml</link>")
			fmt.Fprintln(w, "<atom:link href=\"https://zerm.eu/rss.xml\" rel=\"self\" type=\"application/rss+xml\" />")

			articles := allArticles

			for _, article := range articles {
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
		} else if strings.HasPrefix(r.RequestURI, "/zerm/") {
			var article article
			found := false

			articles := allArticles

			for _, a := range articles {
				if "/zerm/"+a.URL == r.RequestURI {
					article = a
					found = true
					break
				}
			}

			if !found {
				notFound(w, r)
				return
			}

			html, err := getHTMLArticle(r.RequestURI)
			if err != nil {
				internalServerError(w, r, err)
				return
			}

			author, err := readFile("/authors/" + article.Author + ".html")
			if err != nil {
				internalServerError(w, r, err)
				return
			}

			writeHeader(w, r, 200, "", "text/html")

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
		} else if regexp.MustCompile("^/20[0-9]{2}(.html)?$").MatchString(r.RequestURI) {
			s := strings.TrimSuffix(strings.TrimPrefix(r.RequestURI, "/"), ".html")
			year, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				internalServerError(w, r, err)
				return
			}

			writeHeader(w, r, 200, "", "text/html")

			fmt.Fprint(w, "<html><head><title>ZERM GA ", year)
			fmt.Fprint(w, "</title><meta charset='utf-8'/>")
			fmt.Fprint(w, "<link rel='stylesheet' type='text/css' href='style.css'>")
			fmt.Fprint(w, "</head><body>")
			fmt.Fprint(w, "<text class='logo1'>ZERM</text> ")
			fmt.Fprint(w, "<text class='logo2'>ONLINE</text> ")
			fmt.Fprint(w, "<text class='logo1'>G</text>")
			fmt.Fprint(w, "<text class='logo2'>esamt</text> ")
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

			articles := allArticles

			for _, article := range articles {
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
					fmt.Fprint(w, err)
				} else {
					fmt.Fprint(w, "<small>von <strong>")
					w.Write(author)
					fmt.Fprint(w, "</strong></small><br/>")
				}

				fmt.Fprint(w, "<small>", article.Published.Format("02.01.2006 15:04 MST"), "</small></div>")
			}

			fmt.Fprint(w, "</body></html>")
		} else {
			b, err := readFile(r.RequestURI)

			if err != nil {
				notFound(w, r)
				return
			}

			writeHeader(w, r, 200, "", mime.TypeByExtension(r.RequestURI))
			w.Write(b)
		}
	})

	Fatal(http.ListenAndServe(":8099", nil))
}
