package main

import (
	"fmt"
	"net/http"
	"errors"
	"github.com/gocql/gocql"
	"github.com/gin-gonic/gin"
)

var Session *gocql.Session

func init(){
	var err error
	cluster := gocql.NewCluster("127.0.0.1")
	cluster.Keyspace = "library_db"
	Session, err = cluster.CreateSession()
	if err != nil {
		panic(err)
	}
	fmt.Println("CONNECTED TO DATABASE")
}

type book struct {
	ID		 string `json:"id"`
	Title	 string `json:"title"`
	Author	 string `json:"author"`
	Quantity int `json:"quantity"`
}

var books []book

func booksArr(){
	b := map[string]interface{}{}

	iter := Session.Query("select * from books").Iter()
	for iter.MapScan(b){
		books = append(books, book{
			ID: b["id"].(string),
			Title: b["title"].(string),
			Author: b["author"].(string),
			Quantity: b["quantity"].(int),
		})
		b = map[string]interface{}{}
	}

}

func createBook(c *gin.Context){
	var newBook book
	if err := c.BindJSON(&newBook); err != nil {
		return
	}
	if err := Session.Query("insert into books(id, title, author, quantity) values (?, ?, ?, ?)", newBook.ID, newBook.Title, newBook.Author, newBook.Quantity).Exec(); err != nil {
		fmt.Println("Error while insert new book.")
		fmt.Println(err)
		return
	}
	books = append(books, newBook)
	c.IndentedJSON(http.StatusCreated, newBook)
}

func getBooks(c *gin.Context){
	c.IndentedJSON(http.StatusOK, books)
}

func getBookById(id string)(*book, error){

	for i, b := range books {
		if b.ID == id{
			return &books[i], nil
		}
	}

	return nil, errors.New("Book not found.")
}

func bookById(c *gin.Context){
	id := c.Params.ByName("id")
	book, err := getBookById(id)

	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Book not found."})
		return
	}

	c.IndentedJSON(http.StatusOK, book)
}

func checkoutBook(c *gin.Context){
	id, ok := c.GetQuery("id")

	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message":"MIssing id query parameter."})
		return
	}

	book, err := getBookById(id)

	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message":"Book not found."})
		return
	}

	if book.Quantity <= 0 {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message":"Book not available."})
		return
	}

	book.Quantity -= 1
	if err := Session.Query("update books set quantity = ? where id = ?", book.Quantity, id).Exec(); err != nil {
		fmt.Println("Error in update while checkout.")
		fmt.Println(err)
		return
	}
	c.IndentedJSON(http.StatusOK, book)
}

func returnBook(c *gin.Context){
	id, ok := c.GetQuery("id")

	if !ok {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"message":"Missing id query parameter"})
		return
	}

	book, err := getBookById(id)

	if err != nil {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "Book not found"})
		return
	}

	book.Quantity += 1;
	if err := Session.Query("update books set quantity = ? where id = ?", book.Quantity, id).Exec(); err != nil {
		fmt.Println("Error in update while returning book.")
		fmt.Println(err)
		return
	}
	c.IndentedJSON(http.StatusOK, book)
}

func deleteBook(c *gin.Context){
	id := c.Params.ByName("id")

	if err := Session.Query("delete from books where id=?", id).Exec(); err != nil {
		fmt.Println("Error while delete new book.")
		fmt.Println(err)
		return
	}
	j := 0
	for i, b := range books {
		if b.ID == id{
			j = i
			break;
		}
	}
	books[j] = books[len(books)-1]
	books = books[:len(books)-1]

	c.IndentedJSON(http.StatusOK, gin.H{"message":"Delete operation succesfully executed."})

}

func main(){
	booksArr()
	router := gin.Default()
	router.GET("/books", getBooks)
	router.GET("/books/:id", bookById)
	router.POST("/books", createBook)
	router.PATCH("/checkout", checkoutBook) //use query parameter
	router.PATCH("/return", returnBook) //use query parameter
	router.DELETE("/delete/:id", deleteBook)
	router.Run("localhost:3000")
}