package Attendance

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/auth/credentials"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)


const sheetName = "Attendance"

var bot *sheets.Service




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



func Track(spreadsheetID string, studentID int, dateStr string, present bool) error {
	
	if bot == nil {
		return fmt.Errorf("Attendance.Init не был вызван перед Track")
	}
	if spreadsheetID == "" {
		return fmt.Errorf("пустой spreadsheetID")
	}

	
	date, err := time.Parse("02.01.2006", dateStr)
	if err != nil {
		return fmt.Errorf("неверный формат даты %q, нужен день/месяц/год: %w", dateStr, err)
	}

	
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

	fmt.Printf(
		"записано: ID=%d, дата=%s, ячейка=%s, значение=1\n",
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
