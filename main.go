package main

import (
	"AutoTracking/Attendance"
	"AutoTracking/Course"
	"AutoTracking/handler"
	"AutoTracking/logger"
	"log"
	"net/http"
)

func main() {
	// поднимаем логгер первым делом — всё, что ниже, должно писаться и в файл, и в консоль
	if err := logger.Start("Server.log"); err != nil {
		log.Fatalf("не удалось запустить логгер: %v", err)
	}
	// закрываем файл лога при выходе из main (раньше файл закрывался сразу
	// внутри Start(), из-за чего лог в файл переставал работать после старта)
	defer logger.Stop()

	http.HandleFunc("/", handler.Handler)

	if err := Attendance.Init("credentials.json"); err != nil {
		log.Fatalf("не удалось инициализировать Attendance (credentials.json): %v", err)
	}

	if err := Course.Init("config.env"); err != nil {
		log.Fatalf("не удалось инициализировать Course (config.env): %v", err)
	}

	port := "8090"
	log.Printf("сервер запущен на порту %s", port)

	// ListenAndServe блокирует навсегда и возвращает non-nil только при ошибке
	// (например порт уже занят) — раньше эта ошибка нигде не логировалась и терялась
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("сервер упал: %v", err)
	}
}
