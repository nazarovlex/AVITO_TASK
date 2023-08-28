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
	log.Println("Успешное подключение к БД")

	err := createSchema()
	if err != nil {
		log.Fatal("Ошибка создания схемы БД: ", err)
	} else {
		log.Println("Схема БД создана")
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
