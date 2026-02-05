package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/xuri/excelize/v2"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: new -<text>")
		return
	}

	// аргументы
	command := os.Args[1]
	textExercise := strings.Join(os.Args[2:], " ") // всё после первого аргумента

	// проверка команды
	if command != "new" && command != "replace" && command != "add" {
		fmt.Println("Поддерживаются команды: new, replace, add")
		return
	}

	// проверка формата textArg
	if !strings.HasPrefix(textExercise, "-") {
		fmt.Println("Не найден текст. Формат: -<text>")
		return
	}

	text := textExercise[1:] // убираем дефис

	// --- SQLite ---
	log.Println(text)
	db, err := sql.Open("sqlite", "C:\\Work\\LinguisticCombinatorics\\LinguisticCombinatorics\\cmd\\bot\\bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	fmt.Printf(text)
	switch command {
	case "new":
		qRes, err := db.Exec(
			`INSERT INTO Exercise (title) VALUES (?)`,
			text,
		)
		if err != nil {
			log.Fatal(err)
		}
		questionID, _ := qRes.LastInsertId()
		if err != nil {
			log.Printf(string(questionID))
		}
	case "add":
		// --- Excel ---
		f, err := excelize.OpenFile(text + ".xlsx")
		if err != nil {
			log.Fatal(err)
		}

		sheetName := f.GetSheetName(0)
		rows, err := f.GetRows(sheetName)
		if err != nil {
			log.Fatal(err)
		}

		if len(rows) < 2 {
			log.Fatal("недостаточно строк в Excel")
		}

		var currentQuestionID int64
		subQuestionSeq := 1
		for rowIdx, row := range rows {
			if len(row) < 1 {
				continue // пропускаем пустые или короткие строки
			}

			// ---------- Question ----------
			if strings.TrimSpace(row[0]) != "" {
				questionText := strings.TrimSpace(row[0])

				res, err := db.Exec(`INSERT INTO Question (exercise_id, text)
    				VALUES ((SELECT id FROM Exercise WHERE title = ?),?)`, sheetName, questionText)
				if err != nil {
					log.Fatalf("Ошибка вставки Question на строке %d: %v", rowIdx+1, err)
				}
				currentQuestionID, err = res.LastInsertId()
				if err != nil {
					log.Fatal(err)
				}
				subQuestionSeq = 1 // сбрасываем seq_num для нового вопроса
				continue
			}
			// ---------- SubQuestion ----------
			subText := row[1]
			pointing := 0
			if len(row) >= 3 && strings.ToLower(strings.TrimSpace(row[2])) == "true" {
				pointing = 1
			}

			var subQuestionID int64
			if subText != "" {
				res, err := db.Exec(
					`INSERT INTO SubQuestion (question_id, seq_num, pointing, text)
				 VALUES (?, ?, ?, ?)`,
					currentQuestionID, subQuestionSeq, pointing, subText,
				)
				if err != nil {
					log.Fatalf("Ошибка вставки SubQuestion на строке %d: %v", rowIdx+1, err)
				}
				subQuestionID, err = res.LastInsertId()
				if err != nil {
					log.Fatal(err)
				}
				subQuestionSeq++
			}
			// ---------- Options ----------
			if len(row) >= 4 && subQuestionID != 0 {
				for col := 3; col < len(row); col++ {
					optText := strings.TrimSpace(row[col])
					if optText == "" {
						continue
					}

					_, err := db.Exec(
						`INSERT INTO Option (sub_question_id, text)
					 VALUES (?, ?)`,
						subQuestionID, optText)
					if err != nil {
						log.Fatalf("Ошибка вставки Option на строке %d, колонка %d: %v", rowIdx+1, col+1, err)
					}
				}
			}
		}

	}
	fmt.Printf("Новая запись в таблице успешно добавлена")

}
