package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Команды бота и их описания
var commands = map[string]string{
	"start": "Запустить бота",
	"help":  "Помощь по командам",
}

type Item struct {
	Question   string
	QuestionID int64
	Answer     string
	AnswerID   int64
	SeqNum     int
	Prepinanie bool
	Options    map[int64]string
}

func main() {
	// Получаем токен бота из переменных окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("Токен бота не указан. Задайте через TELEGRAM_BOT_TOKEN")
	}
	log.Printf("Токен такой: %s", botToken)
	// Создаём экземпляр бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	bot.Debug = true // Включаем логирование (опционально)

	log.Printf("Бот %s запущен", bot.Self.UserName)

	// Настраиваем канал обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	// Обрабатываем входящие обновления
	for update := range updates {
		if update.CallbackQuery != nil && update.CallbackQuery.Data != "" {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
		if update.Message == nil {
			continue // Игнорируем всё, кроме сообщений
		}

		// Обработка команд
		switch update.Message.Command() {
		case "start":
			handleStartCommand(bot, update.Message)
		case "help":
			handleHelpCommand(bot, update.Message)
		default:
			handleTextMessage(bot, update.Message)
		}
	}
}

// Обработчик команды /start
func handleStartCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Комбинаторика", "Combinatorics"),
			tgbotapi.NewInlineKeyboardButtonData("Аудирование", "Listening"),
		),
	)

	newMsg := tgbotapi.NewMessage(
		msg.Chat.ID,
		"Привет! Я телеграм-бот для практики грамматики татарского языка.\nИспользуй /help для списка команд.",
	)
	newMsg.ReplyMarkup = keyboard

	if _, err := bot.Send(newMsg); err != nil {
		log.Println("Ошибка отправки /start:", err)
	}

}

// Обработчик команды /help
func handleHelpCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	helpText := "Доступные команды:\n"
	for cmd, desc := range commands {
		helpText += "/" + cmd + " - " + desc + "\n"
	}
	sendMessage(bot, msg.Chat.ID, helpText)
}

// Обработчик обычных текстовых сообщений
func handleTextMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	reply := "Я не понимаю твоего сообщения. Попробуй /help"
	sendMessage(bot, msg.Chat.ID, reply)
}

// Утилита для отправки сообщений
func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	_, err := bot.Send(msg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения: %v", err)
	}
}

// Обработчик нажатия кнопок
func handleCallbackQuery(bot *tgbotapi.BotAPI, CallbackQuery *tgbotapi.CallbackQuery) {
	callbackID := CallbackQuery.ID
	msgID := CallbackQuery.Message.MessageID
	chatID := CallbackQuery.Message.Chat.ID
	msgText := CallbackQuery.Message.Text
	//выбрали раздел комбинаторика
	if CallbackQuery.Data == "Combinatorics" {
		LevelsList(bot, CallbackQuery.Message)
	}
	//выбрали упражнение
	if strings.HasPrefix(CallbackQuery.Data, "ExerciseID=") {
		re := regexp.MustCompile(`\d+`)
		numStr := re.FindString(CallbackQuery.Data) // возвращает "123"
		ExerciseID, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			log.Printf("Ошибка конвертации: %v", err)
		}
		InitQuestionField(bot, CallbackQuery.Message, ExerciseID)
	}
	//выбрали ответ
	if strings.HasPrefix(CallbackQuery.Data, "ansID=") {
		re := regexp.MustCompile(`\d+`)
		numStr := re.FindString(CallbackQuery.Data) // возвращает "123"
		optionID, err := strconv.ParseInt(numStr, 10, 64)
		if err != nil {
			log.Printf("Ошибка конвертации: %v", err)
		}
		//questions := initData()
		currentAnswer, nextAnswer, currentIsRight, lastSubquestion, lastQuestion, prepinanie, err := ActuallyAnswer(optionID)
		log.Printf(strconv.FormatBool(lastSubquestion)) //для компилятора
		if err != nil {
			log.Printf("не удалось найти вопрос: %v", err)
		}
		if currentIsRight {
			if lastQuestion {
				editText := tgbotapi.EditMessageTextConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    chatID,
						MessageID: msgID,
						//ReplyMarkup: &tempMarkup,
					},
					Text:      fmt.Sprintf("%s%s%s ✅", msgText, currentAnswer.Answer, prepinanie),
					ParseMode: "",
				}
				bot.Send(editText)
				//финальное сообщение
				editText = tgbotapi.EditMessageTextConfig{
					BaseEdit: tgbotapi.BaseEdit{
						ChatID:    chatID,
						MessageID: msgID,
						//ReplyMarkup: &tempMarkup,
					},
					Text:      fmt.Sprintf("Упражнение закончено ✅"),
					ParseMode: "",
				}
				bot.Send(editText)
				LevelsList(bot, CallbackQuery.Message)
			} else {
				if lastSubquestion {
					editText := tgbotapi.EditMessageTextConfig{
						BaseEdit: tgbotapi.BaseEdit{
							ChatID:    chatID,
							MessageID: msgID,
							//ReplyMarkup: &tempMarkup,
						},
						Text:      fmt.Sprintf("%s%s%s ✅", msgText, currentAnswer.Answer, prepinanie),
						ParseMode: "",
					}
					bot.Send(editText)
					//новое поле кнопок
					InlineKeyboardButtonArray := []tgbotapi.InlineKeyboardButton{}
					keyboard := [][]tgbotapi.InlineKeyboardButton{}
					lineSize := 3
					lineX := 1
					for key, ans := range nextAnswer.Options {
						InlineKeyboardButtonArray = append(InlineKeyboardButtonArray, tgbotapi.NewInlineKeyboardButtonData(ans, fmt.Sprintf("ansID=%d;", key)))
						//log.Printf("кнопка: %v", fmt.Sprintf("ansID=%d;", key))
						if lineX == lineSize {
							keyboard = append(keyboard, InlineKeyboardButtonArray)
							InlineKeyboardButtonArray = []tgbotapi.InlineKeyboardButton{}
							lineX = 0
						}
						lineX++
					}
					keyboard = append(keyboard, InlineKeyboardButtonArray)

					newMsg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Переведите предложение:\n%s \nПеревод: ", nextAnswer.Question))
					tempMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
					newMsg.ReplyMarkup = &tempMarkup
					bot.Send(newMsg)
				} else {
					firstQuestions := nextAnswer
					//log.Printf("Вариант ответа: %v", firstQuestions.Options[1])
					InlineKeyboardButtonArray := []tgbotapi.InlineKeyboardButton{}
					keyboard := [][]tgbotapi.InlineKeyboardButton{}
					lineSize := 3
					lineX := 1
					for key, ans := range firstQuestions.Options {
						InlineKeyboardButtonArray = append(InlineKeyboardButtonArray, tgbotapi.NewInlineKeyboardButtonData(ans, fmt.Sprintf("ansID=%d;", key)))
						//log.Printf("кнопка: %v", fmt.Sprintf("ansID=%d;", key))
						if lineX == lineSize {
							keyboard = append(keyboard, InlineKeyboardButtonArray)
							InlineKeyboardButtonArray = []tgbotapi.InlineKeyboardButton{}
							lineX = 0
						}
						lineX++
					}
					keyboard = append(keyboard, InlineKeyboardButtonArray)
					tempMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
					//newMsg.ReplyMarkup = &tempMarkup
					if prepinanie == " " {
						prepinanie = "\u00A0"
					}
					editText := tgbotapi.EditMessageTextConfig{
						BaseEdit: tgbotapi.BaseEdit{
							ChatID:      chatID,
							MessageID:   msgID,
							ReplyMarkup: &tempMarkup,
						},
						Text:      fmt.Sprintf("%s%s%s", msgText, currentAnswer.Answer, prepinanie),
						ParseMode: "",
					}
					log.Printf("msgText: %s", msgText)
					log.Printf("Answer: %s", currentAnswer.Answer)
					fmt.Printf("prepinanie = %q, len = %d\n", prepinanie, len(prepinanie))
					log.Printf(fmt.Sprintf("%s%s%s", msgText, prepinanie, currentAnswer.Answer))
					bot.Send(editText)
				}
			}

		} else {
			firstQuestions := currentAnswer
			//log.Printf("ответ неверный: %v", firstQuestions.Options[optionID])
			//FirstOptions := FirstQuestions.Options
			InlineKeyboardButtonArray := []tgbotapi.InlineKeyboardButton{}
			keyboard := [][]tgbotapi.InlineKeyboardButton{}
			lineSize := 3
			lineX := 1
			for key, ans := range firstQuestions.Options {
				if key == optionID {
					InlineKeyboardButtonArray = append(InlineKeyboardButtonArray, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("❌ %s", ans), fmt.Sprintf("ansID=%d;", key)))
				} else {
					InlineKeyboardButtonArray = append(InlineKeyboardButtonArray, tgbotapi.NewInlineKeyboardButtonData(ans, fmt.Sprintf("ansID=%d;", key)))
				}
				//log.Printf("кнопка: %v", fmt.Sprintf("ansID=%d;", key))
				if lineX == lineSize {
					keyboard = append(keyboard, InlineKeyboardButtonArray)
					InlineKeyboardButtonArray = []tgbotapi.InlineKeyboardButton{}
					lineX = 0
				}
				lineX++
			}
			keyboard = append(keyboard, InlineKeyboardButtonArray)
			tempMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
			//newMsg.ReplyMarkup = &tempMarkup

			editText := tgbotapi.EditMessageTextConfig{
				BaseEdit: tgbotapi.BaseEdit{
					ChatID:      chatID,
					MessageID:   msgID,
					ReplyMarkup: &tempMarkup,
				},
				Text:      fmt.Sprintf(msgText),
				ParseMode: "",
			}
			bot.Send(editText)
		}
	}

	answerCbq := tgbotapi.CallbackConfig{
		CallbackQueryID: callbackID,
		Text:            "Обработка выполнена",
		ShowAlert:       false,
	}
	bot.Request(answerCbq)
}

// сформировать форму упражения
func InitQuestionField(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, ExerciseID int64) {

	firstQuestions := LoadItem(ExerciseID)
	//FirstOptions := FirstQuestions.Options
	InlineKeyboardButtonArray := []tgbotapi.InlineKeyboardButton{}
	keyboard := [][]tgbotapi.InlineKeyboardButton{}
	lineSize := 3
	lineX := 1
	for key, ans := range firstQuestions.Options {
		InlineKeyboardButtonArray = append(InlineKeyboardButtonArray, tgbotapi.NewInlineKeyboardButtonData(ans, fmt.Sprintf("ansID=%d;", key)))
		//log.Printf("кнопка: %v", fmt.Sprintf("ansID=%d;", key))
		if lineX == lineSize {
			keyboard = append(keyboard, InlineKeyboardButtonArray)
			InlineKeyboardButtonArray = []tgbotapi.InlineKeyboardButton{}
			lineX = 0
		}
		lineX++
	}
	keyboard = append(keyboard, InlineKeyboardButtonArray)

	newMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Переведите предложение:\n%s \nПеревод: ", firstQuestions.Question))
	tempMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	newMsg.ReplyMarkup = &tempMarkup
	bot.Send(newMsg)
}

// получить текущий следующий вопрос и признак правильного ответа
func ActuallyAnswer(optionID int64) (current *Item, next *Item, currentIsRight bool, lastSubquestion bool, lastQuestion bool, prepinanie string, err error) {
	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	const queryItem = `WITH current AS (
    SELECT
        1 AS sort_order,
        'current' AS row_type,

        q.id    AS question_id,
        q.text  AS question_text,

        sq.id   AS subquestion_id,
        sq.text AS subquestion_text,
        sq.seq_num,

        CASE WHEN o.text = sq.text THEN 1 ELSE 0 END AS currentIsRight,

        CASE
            WHEN sq.seq_num = (
                SELECT MAX(seq_num)
                FROM SubQuestion
                WHERE question_id = q.id
            )
            THEN 1 ELSE 0
        END AS lastSubquestion,
		NULL AS prepinanie
    FROM Option o
    JOIN SubQuestion sq ON sq.id = o.sub_question_id
    JOIN Question q ON q.id = sq.question_id
    WHERE o.id = ?
	),

	next_step AS (
    -- следующий SubQuestion
    SELECT
    2 AS sort_order,
    'next' AS row_type,

    q.id    AS question_id,
    q.text  AS question_text,

    sq.id   AS subquestion_id,
    sq.text AS subquestion_text,
    sq.seq_num,

    NULL AS currentIsRight,
    NULL AS lastSubquestion,

   CASE
    WHEN EXISTS (
        SELECT 1
        FROM SubQuestion sq1
        WHERE sq1.question_id = c.question_id
          AND sq1.seq_num = c.seq_num + 1
          AND sq1.pointing = 1
    )
    THEN (
        SELECT sq1.text
        FROM SubQuestion sq1
        WHERE sq1.question_id = c.question_id
          AND sq1.seq_num = c.seq_num + 1
          AND sq1.pointing = 1
        LIMIT 1
    )
    ELSE ''
	END AS prepinanie

	FROM current c
	JOIN SubQuestion sq
    ON sq.question_id = c.question_id
   AND sq.seq_num = (
        CASE
            WHEN EXISTS (
                SELECT 1
                FROM SubQuestion sq1
                WHERE sq1.question_id = c.question_id
                  AND sq1.seq_num = c.seq_num + 1
                  AND sq1.pointing = 1
            )
            THEN c.seq_num + 2
            ELSE c.seq_num + 1
        END
   )
	JOIN Question q ON q.id = sq.question_id
	WHERE c.lastSubquestion = 0


    UNION ALL

    -- первый SubQuestion следующего Question
    SELECT
        2 AS sort_order,
        'next' AS row_type,

        q2.id   AS question_id,
        q2.text AS question_text,

        sq2.id  AS subquestion_id,
        sq2.text AS subquestion_text,
        sq2.seq_num,

        NULL AS currentIsRight,
        NULL AS lastSubquestion,
		NULL AS prepinanie
    FROM current c
    JOIN Question q2
        ON q2.exercise_id = (
            SELECT exercise_id
            FROM Question
            WHERE id = c.question_id
        )
       AND q2.id > c.question_id
    JOIN SubQuestion sq2 ON sq2.question_id = q2.id
    WHERE c.lastSubquestion = 1
      AND sq2.seq_num = (
          SELECT MIN(seq_num)
          FROM SubQuestion
          WHERE question_id = q2.id
      )
    ORDER BY q2.id
    LIMIT 1
	)

	SELECT 
		question_id,
		question_text,
		subquestion_id,
		subquestion_text,
		seq_num,
		currentIsRight,
		lastSubquestion,
		COALESCE(prepinanie, '') AS prepinanie
	FROM (
		SELECT * FROM current
		UNION ALL
		SELECT * FROM next_step
	)
	ORDER BY sort_order;`
	// SQL-запрос
	rows, err := db.Query(queryItem, optionID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	current = &Item{
		Options: make(map[int64]string),
	}

	if rows.Next() {
		var dummyPrepinanie sql.NullString
		if err := rows.Scan(
			&current.QuestionID,
			&current.Question,
			&current.AnswerID,
			&current.Answer,
			&current.SeqNum,
			&currentIsRight,
			&lastSubquestion,
			&dummyPrepinanie); err != nil {
			log.Fatal(err)
		}
		const queryOptions = `SELECT id, text
		FROM Option
		WHERE sub_question_id = ?
		ORDER BY id;`
		rowsOptions, err := db.Query(queryOptions, current.AnswerID)
		if err != nil {
			log.Fatal(err)
		}
		defer rowsOptions.Close()

		for rowsOptions.Next() {
			var optID int64
			var text string

			if err := rowsOptions.Scan(&optID, &text); err != nil {
				log.Fatal(err)
			}

			current.Options[optID] = text
		}
	}

	next = &Item{
		Options: make(map[int64]string),
	}
	if rows.Next() {
		var dummyRight sql.NullBool
		var dummyLast sql.NullBool
		if err := rows.Scan(
			&next.QuestionID,
			&next.Question,
			&next.AnswerID,
			&next.Answer,
			&next.SeqNum,
			&dummyRight,
			&dummyLast,
			&prepinanie); err != nil {
			log.Fatal(err)
		}
		const queryOptions = `SELECT id, text
		FROM Option
		WHERE sub_question_id = ?
		ORDER BY id;`
		rowsOptions, err := db.Query(queryOptions, next.AnswerID)
		if err != nil {
			log.Fatal(err)
		}
		defer rowsOptions.Close()

		for rowsOptions.Next() {
			var optID int64
			var text string

			if err := rowsOptions.Scan(&optID, &text); err != nil {
				log.Fatal(err)
			}

			next.Options[optID] = text
		}
	} else {
		lastQuestion = true
	}

	return current, next, currentIsRight, lastSubquestion, lastQuestion, prepinanie, nil
}

// вывести список упражнений
func LevelsList(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// SQL-запрос
	rows, err := db.Query(`SELECT title,ID FROM Exercise`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var title string
	var id int

	InlineKeyboardButtonArray := []tgbotapi.InlineKeyboardButton{}
	keyboard := [][]tgbotapi.InlineKeyboardButton{}
	for rows.Next() {
		if err := rows.Scan(&title, &id); err != nil {
			log.Fatal(err)
		}
		InlineKeyboardButtonArray = append(InlineKeyboardButtonArray, tgbotapi.NewInlineKeyboardButtonData(title, fmt.Sprintf("ExerciseID=%d;", id)))
		keyboard = append(keyboard, InlineKeyboardButtonArray)
		InlineKeyboardButtonArray = []tgbotapi.InlineKeyboardButton{}
	}
	keyboard = append(keyboard, InlineKeyboardButtonArray)
	newMsg := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Выберите упражнение"))
	tempMarkup := tgbotapi.NewInlineKeyboardMarkup(keyboard...)
	newMsg.ReplyMarkup = &tempMarkup
	bot.Send(newMsg)

}

// получить первый вопрос из упражнения
func LoadItem(ExerciseID int64) Item {
	db, err := sql.Open("sqlite", "bot.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	data := Item{
		Options: make(map[int64]string),
	}

	const queryItem = `
		SELECT
			q.text        AS Question,
			q.id          AS QuestionID,
			sq.text       AS Answer,
			sq.id         AS AnswerID,
			sq.seq_num    AS SeqNum,
			sq.pointing   AS Prepinanie
		FROM Question q
		JOIN SubQuestion sq ON sq.question_id = q.id
		WHERE q.exercise_id = ?
		AND q.id = (
			SELECT id
			FROM Question
			WHERE exercise_id = ?
			ORDER BY id
			LIMIT 1
		)
		AND sq.seq_num = (
			SELECT seq_num
			FROM SubQuestion
			WHERE question_id = q.id
			ORDER BY seq_num
			LIMIT 1
		);`

	row := db.QueryRow(queryItem, ExerciseID, ExerciseID)

	err = row.Scan(
		&data.Question,
		&data.QuestionID,
		&data.Answer,
		&data.AnswerID,
		&data.SeqNum,
		&data.Prepinanie,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatal(err)
		}
		log.Fatal(err)
	}

	const queryOptions = `SELECT id, text
		FROM Option
		WHERE sub_question_id = ?
		ORDER BY id;`
	rows, err := db.Query(queryOptions, data.AnswerID)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var optID int64
		var text string

		if err := rows.Scan(&optID, &text); err != nil {
			log.Fatal(err)
		}

		data.Options[optID] = text
	}

	return data
}
