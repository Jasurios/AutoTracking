package main

import (
	"AutoTracking/Attendance"
	"AutoTracking/Course"
	"AutoTracking/handler"
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler.Handler)

	if err := Attendance.Init("credentials.json"); err != nil {
		log.Fatal(err)
	}

	
	if err := Course.Init("config.env"); err != nil {
		log.Fatal(err)
	}

	port := "8090"
	fmt.Println("Сервер запущен на порту", port)
	http.ListenAndServe(":"+port, nil)
}
