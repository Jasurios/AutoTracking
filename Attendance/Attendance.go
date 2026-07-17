package Attendance

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"cloud.google.com/go/auth/credentials"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const sheetName = "Attendance"

// bot — общий клиент Google Sheets на весь пакет, поднимается один раз в Init
var bot *sheets.Service

// Init поднимает клиент Google Sheets из сервис-аккаунта (credentials.json).
// Дергать один раз при старте сервера, до всех Track().
func Init(credentialsPath string) error {
	raw, err := os.ReadFile(credentialsPath)
	if err != nil {
		return fmt.Errorf("не удалось прочитать credentials %s: %w", credentialsPath, err)
	}

	ctx := context.Background()

	creds, err := credentials.DetectDefault(&credentials.DetectOptions{
		CredentialsJSON: raw,
		Scopes:          []string{sheets.SpreadsheetsScope},
	})
	if err != nil {
		return fmt.Errorf("не удалось разобрать credentials: %w", err)
	}

	srv, err := sheets.NewService(ctx, option.WithAuthCredentials(creds))
	if err != nil {
		return fmt.Errorf("не удалось создать клиент sheets: %w", err)
	}

	bot = srv
	return nil
}

// Track отмечает посещаемость studentID на дату dateStr в таблице spreadsheetID.
// present=true  -> ставит 1 в ячейку студента и красит её зелёным
// present=false -> проставляет -1 всем пустым ячейкам колонки этой даты (розовым),
//
//	то есть массово помечает "отсутствовал" тех, кого не отметили
func Track(spreadsheetID string, studentID int, dateStr string, present bool) error {
	// защита от вызова не в том порядке — Init должен отработать раньше
	if bot == nil {
		return fmt.Errorf("Attendance.Init не был вызван перед Track")
	}
	if spreadsheetID == "" {
		return fmt.Errorf("пустой spreadsheetID")
	}

	// дата приходит строкой вида "02.01.2006" из handler'а
	date, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return fmt.Errorf("неверный формат даты %q, нужен день/месяц/год: %w", dateStr, err)
	}

	// тащим весь лист разом — дешевле, чем дёргать Sheets API по частям
	resp, err := bot.Spreadsheets.Values.Get(spreadsheetID, sheetName).
		ValueRenderOption("UNFORMATTED_VALUE").
		DateTimeRenderOption("SERIAL_NUMBER").
		Do()
	if err != nil {
		return fmt.Errorf("не удалось прочитать лист: %w", err)
	}

	if len(resp.Values) < 3 {
		return fmt.Errorf("на листе нет строки с датами (строка 3)")
	}

	// строка 3 (индекс 2) — это даты в виде excel serial number,
	// ищем колонку с нужной датой; сравниваем с допуском 0.5, т.к. в таблице
	// это float и могут быть погрешности округления
	dateRow := resp.Values[2]
	targetSerial := excelSerial(date)
	colIdx := -1

	for i := 2; i < len(dateRow); i++ {
		cellVal, ok := dateRow[i].(float64)
		if !ok {
			continue
		}

		if diff := cellVal - targetSerial; diff > -0.5 && diff < 0.5 {
			colIdx = i
			break
		}
	}

	if colIdx == -1 {
		return fmt.Errorf("дата %s не найдена в таблице", dateStr)
	}

	// ветка "отсутствовал": никого конкретно не отмечаем, а массово
	// проставляем -1 всем, кто ещё пустой в этой колонке на эту дату
	if !present {
		spreadsheet, err := bot.Spreadsheets.Get(spreadsheetID).Do()
		if err != nil {
			return fmt.Errorf("ошибка получения таблицы: %w", err)
		}

		var sheetID int64

		for _, s := range spreadsheet.Sheets {
			if s.Properties.Title == sheetName {
				sheetID = s.Properties.SheetId
				break
			}
		}

		var requests []*sheets.Request

		for i := 3; i < len(resp.Values); i++ {
			isEmpty := len(resp.Values[i]) <= colIdx ||
				resp.Values[i][colIdx] == nil ||
				resp.Values[i][colIdx] == ""

			if isEmpty {
				requests = append(requests,
					&sheets.Request{
						UpdateCells: &sheets.UpdateCellsRequest{
							Range: &sheets.GridRange{
								SheetId:          sheetID,
								StartRowIndex:    int64(i),
								EndRowIndex:      int64(i + 1),
								StartColumnIndex: int64(colIdx),
								EndColumnIndex:   int64(colIdx + 1),
							},
							Rows: []*sheets.RowData{
								{
									Values: []*sheets.CellData{
										{
											UserEnteredValue: &sheets.ExtendedValue{
												NumberValue: func() *float64 {
													v := float64(-1)
													return &v
												}(),
											},
											UserEnteredFormat: &sheets.CellFormat{
												BackgroundColor: &sheets.Color{
													Red:   0.95,
													Green: 0.75,
													Blue:  0.75,
												},
											},
										},
									},
								},
							},
							Fields: "userEnteredValue,userEnteredFormat.backgroundColor",
						},
					},
				)
			}
		}

		if len(requests) > 0 {
			_, err = bot.Spreadsheets.BatchUpdate(
				spreadsheetID,
				&sheets.BatchUpdateSpreadsheetRequest{
					Requests: requests,
				},
			).Do()

			if err != nil {
				return fmt.Errorf("ошибка записи 0 и покраски пустых ячеек: %w", err)
			}
		}

		return nil
	}

	// ветка "присутствовал": ищем строку конкретного studentID (колонка A, с 4-й строки)
	rowIdx := -1

	for i := 3; i < len(resp.Values); i++ {
		row := resp.Values[i]

		if len(row) == 0 {
			continue
		}

		idVal, ok := row[0].(float64)
		if !ok {
			continue
		}

		if int(idVal) == studentID {
			rowIdx = i
			break
		}
	}

	if rowIdx == -1 {
		return fmt.Errorf("ID %d не найден в таблице", studentID)
	}

	// нашли строку — считаем адрес ячейки (буква колонки + номер строки) и пишем 1
	value := 1

	cellAddr := fmt.Sprintf(
		"%s!%s%d",
		sheetName,
		columnLetter(colIdx),
		rowIdx+1,
	)

	_, err = bot.Spreadsheets.Values.Update(
		spreadsheetID,
		cellAddr,
		&sheets.ValueRange{
			Values: [][]interface{}{{value}},
		},
	).
		ValueInputOption("RAW").
		Do()

	if err != nil {
		return fmt.Errorf("ошибка записи в ячейку %s: %w", cellAddr, err)
	}

	// sheetID нужен отдельно от spreadsheetID — это internal id листа "Attendance"
	// внутри таблицы, Values.Update его не принимает, а BatchUpdate требует
	spreadsheet, err := bot.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("ошибка получения таблицы: %w", err)
	}

	var sheetID int64

	for _, s := range spreadsheet.Sheets {
		if s.Properties.Title == sheetName {
			sheetID = s.Properties.SheetId
			break
		}
	}

	// красим ячейку зелёным — чисто визуальная отметка "присутствовал"
	_, err = bot.Spreadsheets.BatchUpdate(
		spreadsheetID,
		&sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				{
					RepeatCell: &sheets.RepeatCellRequest{
						Range: &sheets.GridRange{
							SheetId:          sheetID,
							StartRowIndex:    int64(rowIdx),
							EndRowIndex:      int64(rowIdx + 1),
							StartColumnIndex: int64(colIdx),
							EndColumnIndex:   int64(colIdx + 1),
						},
						Cell: &sheets.CellData{
							UserEnteredFormat: &sheets.CellFormat{
								BackgroundColor: &sheets.Color{
									Red:   0.75,
									Green: 0.90,
									Blue:  0.75,
								},
							},
						},
						Fields: "userEnteredFormat.backgroundColor",
					},
				},
			},
		},
	).Do()

	if err != nil {
		return fmt.Errorf("ошибка установки цвета ячейки %s: %w", cellAddr, err)
	}

	// пишем через log, а не fmt.Printf, чтобы это тоже уходило в Server.log,
	// а не только в консоль
	log.Printf(
		"записано: ID=%d, дата=%s, ячейка=%s, значение=1",
		studentID,
		dateStr,
		cellAddr,
	)

	return nil
}

func excelSerial(t time.Time) float64 {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	return t.UTC().Sub(epoch).Hours() / 24
}

func columnLetter(idx int) string {
	letter := ""
	idx++
	for idx > 0 {
		idx--
		letter = string(rune('A'+idx%26)) + letter
		idx /= 26
	}
	return letter
}
