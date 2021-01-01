package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gomarkdown/markdown"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(400)
			fmt.Fprintln(w, "WTF are you tryin to achieve?! This is GET only.")
			return
		}
		file, err := os.OpenFile("README.md", os.O_RDONLY, 0o644)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, err)
			return
		}
		defer file.Close()
		stat, err := file.Stat()
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintln(w, err)
			return
		}
		md := make([]byte, stat.Size())
		//TODO: err handling
		file.Read(md)
		w.Write(markdown.ToHTML(md, nil, nil))
	})

	http.ListenAndServe(":8099", nil)
}
