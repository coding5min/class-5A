package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Student struct {
	ID        int    `json:"id"`
	Fullname  string `json:"fullname"`
	Age       int    `json:"age"`
	ClassName string `json:"class_name"`
	Phone     string `json:"phone"`
	CreatedAt string `json:"created_at"`
}

type StudentInput struct {
	Fullname  string `json:"fullname" binding:"required"`
	Age       int    `json:"age" binding:"required"`
	ClassName string `json:"class_name"`
	Phone     string `json:"phone"`
}

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println(".env файл не найден, читаем переменные окружения из системы")
	}

	db = connectDB()
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("PostgreSQL не отвечает:", err)
	}

	log.Println("PostgreSQL подключен")

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Go Gin Students API работает",
		})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	r.GET("/students", getStudents)
	r.GET("/students/:id", getStudentByID)
	r.POST("/students", createStudent)
	r.PUT("/students/:id", updateStudent)
	r.DELETE("/students/:id", deleteStudent)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Сервер запущен на порту:", port)

	err = r.Run("0.0.0.0:" + port)
	if err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}

func connectDB() *sql.DB {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")
	sslmode := os.Getenv("DB_SSLMODE")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host,
		port,
		user,
		password,
		dbname,
		sslmode,
	)

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("Ошибка подключения к PostgreSQL:", err)
	}

	return database
}

func getStudents(c *gin.Context) {
	rows, err := db.Query(`
		SELECT 
			id, 
			fullname, 
			age, 
			class_name, 
			COALESCE(phone, ''), 
			created_at
		FROM students
		ORDER BY id DESC
	`)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка получения списка учеников",
		})
		return
	}

	defer rows.Close()

	students := []Student{}

	for rows.Next() {
		var student Student

		err := rows.Scan(
			&student.ID,
			&student.Fullname,
			&student.Age,
			&student.ClassName,
			&student.Phone,
			&student.CreatedAt,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Ошибка чтения данных",
			})
			return
		}

		students = append(students, student)
	}

	c.JSON(http.StatusOK, students)
}

func getStudentByID(c *gin.Context) {
	id := c.Param("id")

	var student Student

	err := db.QueryRow(`
		SELECT 
			id, 
			fullname, 
			age, 
			class_name, 
			COALESCE(phone, ''), 
			created_at
		FROM students
		WHERE id = $1
	`, id).Scan(
		&student.ID,
		&student.Fullname,
		&student.Age,
		&student.ClassName,
		&student.Phone,
		&student.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ученик не найден",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка получения ученика",
		})
		return
	}

	c.JSON(http.StatusOK, student)
}

func createStudent(c *gin.Context) {
	var input StudentInput

	err := c.ShouldBindJSON(&input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Введите fullname и age",
		})
		return
	}

	if input.ClassName == "" {
		input.ClassName = "5A"
	}

	var student Student

	err = db.QueryRow(`
		INSERT INTO students 
			(fullname, age, class_name, phone)
		VALUES 
			($1, $2, $3, $4)
		RETURNING 
			id, fullname, age, class_name, COALESCE(phone, ''), created_at
	`,
		input.Fullname,
		input.Age,
		input.ClassName,
		input.Phone,
	).Scan(
		&student.ID,
		&student.Fullname,
		&student.Age,
		&student.ClassName,
		&student.Phone,
		&student.CreatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка добавления ученика",
		})
		return
	}

	c.JSON(http.StatusCreated, student)
}

func updateStudent(c *gin.Context) {
	id := c.Param("id")

	var input StudentInput

	err := c.ShouldBindJSON(&input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Неверные данные",
		})
		return
	}

	if input.ClassName == "" {
		input.ClassName = "5A"
	}

	var student Student

	err = db.QueryRow(`
		UPDATE students
		SET 
			fullname = $1,
			age = $2,
			class_name = $3,
			phone = $4
		WHERE id = $5
		RETURNING 
			id, fullname, age, class_name, COALESCE(phone, ''), created_at
	`,
		input.Fullname,
		input.Age,
		input.ClassName,
		input.Phone,
		id,
	).Scan(
		&student.ID,
		&student.Fullname,
		&student.Age,
		&student.ClassName,
		&student.Phone,
		&student.CreatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ученик не найден",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка редактирования ученика",
		})
		return
	}

	c.JSON(http.StatusOK, student)
}

func deleteStudent(c *gin.Context) {
	id := c.Param("id")

	result, err := db.Exec(`
		DELETE FROM students
		WHERE id = $1
	`, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка удаления ученика",
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Ошибка проверки удаления",
		})
		return
	}

	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ученик не найден",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ученик удалён",
	})
}