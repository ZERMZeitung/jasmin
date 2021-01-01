package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
)

func internalServerError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, err)
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "WTF are you tryin to achieve?! This is GET only.")
			return
		}

		path := r.RequestURI

		if strings.HasPrefix(path, "/zerm") {
			//so far havent managed to exploit, but it should work in theory:
			//echo 'GET /zerm/../../../../etc/passwd HTTP/1.1
			//Host: localhost:8099
			//User-Agent: curl/7.64.1
			//Accept: */*
			//' |nc localhost 8099
			//TODO: check this
			path = fmt.Sprintf("../zerm.eu%s", path)

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
		}
	})

	http.ListenAndServe(":8099", nil)
}
