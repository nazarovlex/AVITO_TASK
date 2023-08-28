package main

import (
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

func main() {
	initDB()
	router := httprouter.New()
	router.GET("/users", getUsers)
	router.POST("/users", createUser)
	router.GET("/users/:id", getUser)
	router.PUT("/users/:id", updateUser)
	router.DELETE("/users/:id", deleteUser)

	router.GET("/segments", getSegments)
	router.POST("/segments", createSegment)
	router.PUT("/segments/:slug", updateSegment)
	router.DELETE("/segments/:slug", deleteSegment)

	router.POST("/user_segments", addSegmentsToUser)

	log.Println("Сервер запущен на порту :8000")
	err := http.ListenAndServe(":8000", router)
	if err != nil {
		log.Fatal(err)
	}

}
