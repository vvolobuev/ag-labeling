package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {

	http.HandleFunc("/", fileHandler)

	fmt.Println("Файловый сервер запущен на порту 8081...")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		fmt.Fprintf(w, "Файловый сервер работает!")
		return
	}

	filename := r.URL.Path[1:]
	filePath := filepath.Join(".", filename)

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}
	defer file.Close()

	contentType := "application/octet-stream"
	switch filepath.Ext(filename) {
	case ".webp":
		contentType = "image/webp"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")

	io.Copy(w, file)
}
