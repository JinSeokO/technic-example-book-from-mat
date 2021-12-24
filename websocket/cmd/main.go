package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

type templateHandler struct {
	once     sync.Once
	fileName string
	templ    *template.Template
}

func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			w.WriteHeader(500)
			if _, err := w.Write([]byte(err.Error())); err != nil {
				log.Fatal(err)
			}
		}
		t.templ = template.Must(template.ParseFiles(filepath.Join(dir, "websocket", "templates", t.fileName)))
	})
	t.templ.Execute(w, nil)
}

func main() {
	http.Handle("/", &templateHandler{fileName: "chat.html"})
	// 웹 서버 시작
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalln(err)
	}
}
