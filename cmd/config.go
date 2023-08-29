package main

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"log"
	"time"
)

type User struct {
	ID       int                  `json:"id"`
	Name     string               `json:"name"`
	Segments map[string]time.Time `json:"segments"`
}

type Segment struct {
	ID          int               `json:"id"`
	Name        string            `json:"name" pg:",unique"`
	Description string            `json:"description"`
	Users       map[int]time.Time `json:"users"`
}

type AddSegmentRequest struct {
	UserID          int            `json:"user_id"`
	SegmentsToAdd   map[string]int `json:"segments_to_add"`
	SegmentToDelete []string       `json:"segment_to_delete"`
	Override        bool           `json:"override"`
}

var db *pg.DB

func initDB() {
	opts := &pg.Options{
		User:     "postgres",
		Password: "postgres",
		Addr:     "localhost:5432",
		Database: "test_api",
	}

	db = pg.Connect(opts)
	if db == nil {
		log.Fatal("Ошибка подключения к БД")
	}
	defer func(db *pg.DB) {
		err := db.Close()
		if err != nil {

		}
	}(db)

	ctx := context.Background()
	if err := db.Ping(ctx); err != nil {
		log.Fatal("DB connection error:", err)
	}
	log.Println("Successful connection to DB")

	err := createSchema()
	if err != nil {
		log.Fatal("Create DB schemas error: ", err)
	} else {
		log.Println("DB schemas created")
	}
}

func createSchema() error {
	models := []interface{}{
		(*User)(nil),
		(*Segment)(nil),
	}

	for _, model := range models {
		err := db.Model(model).CreateTable(&orm.CreateTableOptions{
			IfNotExists: true,
			Temp:        false,
		})
		if err != nil {
			return err
		}
	}
	return nil

}
