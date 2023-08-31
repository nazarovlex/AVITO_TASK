package main

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/julienschmidt/httprouter"
	"github.com/nazarovlex/AVITO_TASK/internal/db"
	"github.com/nazarovlex/AVITO_TASK/internal/runner"
	"log"
	"net/http"
)

func main() {
	var pgConn *pg.DB
	opts := &pg.Options{
		User:     "postgres",
		Password: "postgres",
		Addr:     "web:5432",
		Database: "test_api",
	}

	pgConn = pg.Connect(opts)
	sql := db.NewSql(pgConn)
	dbService := db.NewService(sql)
	log.Println("Successful connection to DB")

	ctx := context.Background()

	err := dbService.CreateEnumType(ctx)
	if err != nil {
		log.Fatal("Creating ENUM type error: ", err)
	}

	err = db.CreateSchema(ctx, dbService)
	if err != nil {
		log.Fatal("Create DB schemas error: ", err)
	} else {
		log.Println("DB schemas created")
	}

	err = dbService.CreateIndexes(ctx)
	if err != nil {
		log.Fatal("Create DB indexes error: ", err)
	} else {
		log.Println("DB indexes created")
	}

	go runner.Runner(ctx, dbService)

	serve(ctx, dbService)
}

func serve(ctx context.Context, dbService *db.Service) {
	router := httprouter.New()

	// users routes
	router.GET("/users", getUsers(ctx, dbService))
	router.GET("/users/:id", getUser(ctx, dbService))
	router.POST("/users", createUser(ctx, dbService))
	router.DELETE("/users/:id", deleteUser(ctx, dbService))

	// slugs routes
	router.POST("/segments", createSegment(ctx, dbService))
	router.DELETE("/segments/:slug", deleteSegment(ctx, dbService))
	router.PUT("/segments/:slug", updateSegment(ctx, dbService))

	// add and delete user slugs route
	router.POST("/user_segments", addSegmentsToUser(ctx, dbService))

	// reports save and download
	router.GET("/get_report", createReport(ctx, dbService))
	router.GET("/download_report/:filename", downloadReport)

	log.Println("Server listen and serve on port :8000")
	err := http.ListenAndServe(":8000", router)
	if err != nil {
		log.Fatal(err)
	}
}
