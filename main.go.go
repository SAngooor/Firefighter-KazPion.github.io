package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := "8080" // Можно сменить порт, если нужно
	fmt.Println("Сервер запущен на http://localhost:" + port)

	http.HandleFunc("/", handler)

	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Обработан запрос:", r.Method, r.URL.Path)
	fmt.Fprintln(w, "Hello from Go server!")
}
