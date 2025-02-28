package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	_ "github.com/alexbrainman/odbc" // ODBC-драйвер для MS Access
)

// Структура данных для записи в БД
type SurveyResponse struct {
	Email   string `json:"email"`
	Address string `json:"address"`
	Score   int    `json:"score"`
	Level   string `json:"level"`
}

// Структура запроса для Ollama
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

var db *sql.DB

func main() {
	// Подключение к MS Access
	dbPath := "dbPath := "C:\\Users\\Asus\\Desktop\\Moya papka\\fire.accdb"
	connStr := fmt.Sprintf("Driver={Microsoft Access Driver (*.mdb, *.accdb)};DBQ=%s;", dbPath)

	var err error
	db, err = sql.Open("odbc", connStr)
	if err != nil {
		log.Fatal("❌ Ошибка подключения к БД:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("❌ Ошибка соединения с БД:", err)
	}

	createTable()

	// Настраиваем маршруты
	http.HandleFunc("/submitSurvey", handleSurvey)
	http.HandleFunc("/downloadAccess", downloadAccessHandler)
	http.HandleFunc("/generate", handleGenerate)
	http.HandleFunc("/fire-alert", handleFireAlert)
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		enableCors(&w)
		fmt.Fprintln(w, "✅ Сервер работает!")
	})

	fmt.Println("✅ Сервер запущен на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// 📌 Функция создания таблицы
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS SurveyResults (
		ID AUTOINCREMENT PRIMARY KEY,
		Email TEXT(255) NOT NULL UNIQUE,
		Address TEXT(255) NOT NULL,
		Score INT NOT NULL,
		[Level] TEXT(50) NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Println("❌ Ошибка создания таблицы:", err)
	} else {
		fmt.Println("✅ Таблица SurveyResults успешно создана!")
	}
}

// 📌 Обработчик для тревоги (от Raspberry Pi)
// 📌 Обработчик для тревоги (от Raspberry Pi)
func handleFireAlert(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, `{"error": "Метод запрещён"}`, http.StatusMethodNotAllowed)
		return
	}

	// Получаем последний адрес пожара
	var lastAddress string
	query := "SELECT TOP 1 Address FROM SurveyResults ORDER BY ID DESC"
	err := db.QueryRow(query).Scan(&lastAddress)
	if err != nil {
		log.Println("❌ Ошибка запроса к БД:", err)

		// Отправляем JSON-ошибку вместо обычного текста
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Ошибка получения адреса из БД"})
		return
	}

	// Логируем пожар в консоль
	alertMessage := fmt.Sprintf("🚨 Внимание! Обнаружено возгорание по адресу: %s", lastAddress)
	fmt.Println(alertMessage)

	// Отправляем JSON-ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"address": lastAddress})
}

// 📌 Обработчик для записи данных в Access
func handleSurvey(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "❌ Метод запрещён", http.StatusMethodNotAllowed)
		return
	}

	var survey SurveyResponse
	err := json.NewDecoder(r.Body).Decode(&survey)
	if err != nil {
		http.Error(w, "❌ Ошибка JSON", http.StatusBadRequest)
		return
	}

	if survey.Email == "" || survey.Address == "" {
		http.Error(w, "❌ Поля email и address обязательны", http.StatusBadRequest)
		return
	}

	// Определяем уровень безопасности
	if survey.Score > 75 {
		survey.Level = "Высокий уровень безопасности"
	} else if survey.Score > 50 {
		survey.Level = "Средний уровень безопасности"
	} else {
		survey.Level = "Низкий уровень безопасности"
	}

	// Проверяем, есть ли уже Email
	var existingEmail string
	checkQuery := "SELECT Email FROM SurveyResults WHERE Email = ?"
	err = db.QueryRow(checkQuery, survey.Email).Scan(&existingEmail)
	if err == nil {
		http.Error(w, "⚠ Email уже зарегистрирован!", http.StatusConflict)
		return
	}

	query := "INSERT INTO SurveyResults (Email, Address, Score, [Level]) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(query, survey.Email, survey.Address, survey.Score, survey.Level)
	if err != nil {
		http.Error(w, "❌ Ошибка записи в БД", http.StatusInternalServerError)
		log.Println("Ошибка вставки в БД:", err)
		return
	}

	fmt.Println("✅ Данные сохранены:", survey.Email, survey.Score, survey.Level)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "✅ Данные успешно сохранены!")
}

// 📌 Обработчик для скачивания Access-файла
func downloadAccessHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	filePath := "C:\\Users\\HP\\Desktop\\fire.accdb"

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "❌ Файл не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=fire.accdb")
	w.Header().Set("Content-Type", "application/vnd.ms-access")

	http.ServeFile(w, r, filePath)
}

// 📌 Обработчик запроса к Ollama
func handleGenerate(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Метод запрещён", http.StatusMethodNotAllowed)
		return
	}

	var request OllamaRequest
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Ошибка JSON", http.StatusBadRequest)
		return
	}

	ollamaReq := OllamaRequest{
		Model:  "llama3",
		Prompt: request.Prompt,
		Stream: false,
	}

	reqBody, _ := json.Marshal(ollamaReq)

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		http.Error(w, "Ошибка при запросе к Ollama", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	w.Header().Set("Content-Type", "application/json")
	w.Write(body)
}

// 📌 Включаем CORS
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	(*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

