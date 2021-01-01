package main

import (
	"fmt"
	"net/http"

	"github.com/gomarkdown/markdown"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(400)
			fmt.Fprintln(w, "WTF are you tryin to achieve?! This is GET only.")
			return
		}
		w.Write(markdown.ToHTML([]byte("## Ehre"), nil, nil))
	})

	http.ListenAndServe(":8099", nil)
}
