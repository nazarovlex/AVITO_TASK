package main

import (
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

type UserSegmentHistory struct {
	ID        int       `pg:"id,pk" json:"id"`
	UserID    int       `pg:"user_id" json:"user_id"`
	Slug      string    `pg:"segment" json:"slug"`
	Operation string    `pg:"operation" json:"operation"`
	Timestamp time.Time `pg:"timestamp" json:"timestamp"`
}

type AddSegmentRequest struct {
	UserID          int            `json:"user_id"`
	SegmentsToAdd   map[string]int `json:"segments_to_add"`
	SegmentToDelete []string       `json:"segment_to_delete"`
	Override        bool           `json:"override"`
}

type GetReportRequest struct {
	Year  int `json:"year"`
	Month int `json:"month"`
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
		(*UserSegmentHistory)(nil),
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
