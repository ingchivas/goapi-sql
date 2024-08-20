package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Book struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

var db *sql.DB

func main() {
	initDB()

	createTable()

	r := gin.Default()

	r.GET("/ping", ping)
	r.GET("/books", getBooks)
	r.GET("/books/:id", getBook)
	r.POST("/books", createBook)
	r.PUT("/books/:id", updateBook)
	r.DELETE("/books/:id", deleteBook)
	r.PATCH("/books/:id", patchBook)

	r.Run(":8000")
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./books.db")
	if err != nil {
		log.Fatalf("Failed to connect to the database: %s", err)
	}
}

func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		author TEXT
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatalf("Failed to create table: %s", err)
	}
}

func ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

func getBooks(c *gin.Context) {
	rows, err := db.Query("SELECT id, title, author FROM books")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query books"})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var book Book
		if err := rows.Scan(&book.ID, &book.Title, &book.Author); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan book"})
			return
		}
		books = append(books, book)
	}

	c.JSON(http.StatusOK, books)
}

func getBook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid book ID"})
		return
	}

	stmt, err := db.Prepare("SELECT id, title, author FROM books WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	var book Book
	err = stmt.QueryRow(id).Scan(&book.ID, &book.Title, &book.Author)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query book"})
		}
		return
	}

	c.JSON(http.StatusOK, book)
}

func createBook(c *gin.Context) {
	var newBook Book
	if err := c.BindJSON(&newBook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	stmt, err := db.Prepare("INSERT INTO books (title, author) VALUES (?, ?)")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	result, err := stmt.Exec(newBook.Title, newBook.Author)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert book"})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve last insert ID"})
		return
	}
	newBook.ID = int(id)

	c.JSON(http.StatusCreated, newBook)
}

func updateBook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid book ID"})
		return
	}

	var updatedBook Book
	if err := c.BindJSON(&updatedBook); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	stmt, err := db.Prepare("UPDATE books SET title = ?, author = ? WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(updatedBook.Title, updatedBook.Author, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book"})
		return
	}

	c.JSON(http.StatusOK, updatedBook)
}

func deleteBook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid book ID"})
		return
	}

	stmt, err := db.Prepare("DELETE FROM books WHERE id = ?")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete book"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Book deleted"})
}

func patchBook(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid book ID"})
		return
	}

	var patchData map[string]interface{}
	if err := c.BindJSON(&patchData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	query := "UPDATE books SET "
	args := []interface{}{}

	if title, exists := patchData["title"]; exists {
		query += "title = ?, "
		args = append(args, title)
	}
	if author, exists := patchData["author"]; exists {
		query += "author = ?, "
		args = append(args, author)
	}

	query = query[:len(query)-2]
	query += " WHERE id = ?"
	args = append(args, id)

	stmt, err := db.Prepare(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare statement"})
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update book"})
		return
	}

	var updatedBook Book
	db.QueryRow("SELECT id, title, author FROM books WHERE id = ?", id).Scan(&updatedBook.ID, &updatedBook.Title, &updatedBook.Author)

	c.JSON(http.StatusOK, updatedBook)
}
