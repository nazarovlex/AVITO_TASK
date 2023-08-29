package main

import (
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

func main() {
	initDB()
	router := httprouter.New()

	// users routes
	router.GET("/users", getUsers)
	router.POST("/users", createUser)
	router.GET("/users/:id", getUser)
	router.PUT("/users/:id", updateUser)
	router.DELETE("/users/:id", deleteUser)

	// slugs routes
	router.GET("/segments", getSegments)
	router.POST("/segments", createSegment)
	router.PUT("/segments/:slug", updateSegment)
	router.DELETE("/segments/:slug", deleteSegment)

	// add and delete user slugs route
	router.POST("/user_segments", addSegmentsToUser)

	// reports save and download
	router.GET("/get_report", createReport)
	router.GET("/download_report/:filename", downloadReport)

	log.Println("Server listen and serve on port :8000")
	err := http.ListenAndServe(":8000", router)
	if err != nil {
		log.Fatal(err)
	}

}
