package logger

import (
	"io"
	"log"
	"os"
)

// держим файл на уровне пакета, чтобы можно было закрыть его при выключении
// сервера через Stop() — раньше файл закрывался прямо внутри Start()
var logFile *os.File

// Start открывает лог-файл и дублирует весь log.* и в файл, и в консоль (os.Stdout).
// ВАЖНО: файл специально НЕ закрывается тут через defer — раньше было именно так,
// и из-за этого логгер писал в закрытый файл уже после первого вызова Start()
// (лог в файл переставал работать сразу после старта). Закрывать файл нужно
// вызовом Stop() при завершении программы (например через defer logger.Stop()).
func Start(filename string) error {
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	logFile = f

	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	return nil
}

// Stop закрывает лог-файл. Вызывать через defer сразу после logger.Start в main.
func Stop() {
	if logFile != nil {
		logFile.Close()
	}
}
