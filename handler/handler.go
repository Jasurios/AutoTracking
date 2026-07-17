package handler

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"AutoTracking/Attendance"
	"AutoTracking/Course"
)

// Handler разбирает ссылку вида /<курс>?id=<studentID> и отмечает посещаемость
// в гугл-таблице соответствующего курса. Дергается из QR-кодов (main.py сканит
// QR и просто дёргает эту ссылку GET-запросом).
func Handler(w http.ResponseWriter, r *http.Request) {
	course := strings.TrimPrefix(r.URL.Path, "/")
	idStr := r.URL.Query().Get("id")

	if course == "" || course == "favicon.ico" {
		http.NotFound(w, r)
		return
	}

	spreadsheetID, ok := Course.Choise(course)
	if !ok {
		log.Printf("неизвестный курс: %q", course)
		http.Error(w, "неизвестный курс", http.StatusNotFound)
		return
	}

	studentID, err := strconv.Atoi(idStr)
	if err != nil {
		log.Printf("некорректный id: %q", idStr)
		http.Error(w, "некорректный id", http.StatusBadRequest)
		return
	}

	// текущая дата — по ней ищем нужную колонку в таблице
	today := time.Now().Format("02.01.2006")

	// служебный id "1488" — ручная отметка отсутствия (ставит 0 вместо 1).
	// TODO: вынести такой "секретный" id в config.env, а не хардкодить в коде,
	// плюс само число подозрительно палевное для паблик-репы — стоит сменить.
	if idStr == "1488" {
		if err := Attendance.Track(spreadsheetID, studentID, today, false); err != nil {
			log.Printf("ошибка отметки отсутствия (курс=%s, id=%d): %v", course, studentID, err)
			http.Error(w, "ошибка записи посещаемости", http.StatusInternalServerError)
			return
		}
		log.Printf("отмечено отсутствие: курс=%s, id=%d, дата=%s", course, studentID, today)
		fmt.Fprintf(w, "Отмечено отсутствие: %s, ID %d, %s\n", course, studentID, today)
		return
	}

	if err := Attendance.Track(spreadsheetID, studentID, today, true); err != nil {
		log.Printf("ошибка записи посещаемости (курс=%s, id=%d): %v", course, studentID, err)
		http.Error(w, "ошибка записи посещаемости", http.StatusInternalServerError)
		return
	}

	log.Printf("отмечено: курс=%s, id=%d, дата=%s", course, studentID, today)
	fmt.Fprintf(w, "Отмечено: %s, ID %d, %s\n", course, studentID, today)
}
