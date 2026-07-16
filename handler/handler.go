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

	
	
	today := time.Now().Format("02.01.2006")

	if idStr=="1488" {
		Attendance.Track(spreadsheetID, studentID, today, false)
		return
	}else if err := Attendance.Track(spreadsheetID, studentID, today, true); err != nil {
		log.Printf("ошибка записи посещаемости (курс=%s, id=%d): %v", course, studentID, err)
		http.Error(w, "ошибка записи посещаемости", http.StatusInternalServerError)
		return
	}

	log.Printf("отмечено: курс=%s, id=%d, дата=%s", course, studentID, today)
	fmt.Fprintf(w, "Отмечено: %s, ID %d, %s\n", course, studentID, today)
}
