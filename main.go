package main

import (
	"encoding/json"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
	"time"
)

//func init() {
//	// Register many to many model so ORM can better recognize m2m relation.
//	// This should be done before dependant models are used.
//	orm.RegisterTable((*UserSegments)(nil))
//}

type User struct {
	ID       int                  `json:"id"`
	Name     string               `json:"name"`
	Segments map[string]time.Time `json:"segments"`
}

type Segment struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type addSegmentRequest struct {
	UserID          int            `json:"user_id"`
	SegmentsToAdd   map[string]int `json:"segments_to_add"`
	SegmentToDelete []string       `json:"segment_to_delete"`
	Override        bool           `json:"override"`
}

var db *pg.DB

func main() {
	initDB()
	router := httprouter.New()
	router.GET("/users", listUsers)
	router.POST("/users", createUser)
	router.GET("/users/:id", getUser)
	router.PUT("/users/:id", updateUser)
	router.DELETE("/users/:id", deleteUser)

	router.GET("/segments", listSegments)
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

func addSegmentsToUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var requestData addSegmentRequest
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	var currentUser User
	err = db.Model(&currentUser).Where("id=?", requestData.UserID).Select()
	if err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	// map init
	if currentUser.Segments == nil {
		currentUser.Segments = make(map[string]time.Time)
	}

	// Delete segments
	for _, segment := range requestData.SegmentToDelete {
		delete(currentUser.Segments, segment)
	}

	// Add new segments with TTL
	for segment, ttl := range requestData.SegmentsToAdd {
		if _, ok := currentUser.Segments[segment]; ok && requestData.Override {
			currentUser.Segments[segment] = time.Now().Add(time.Duration(ttl) * time.Hour)
		} else if _, ok := currentUser.Segments[segment]; !ok {
			currentUser.Segments[segment] = time.Now().Add(time.Duration(ttl) * time.Hour)
		}
	}

	// Update user in DB
	_, err = db.Model(&currentUser).WherePK().Update()
	if err != nil {
		http.Error(w, "Ошибка добавления сегментов пользователю", http.StatusBadRequest)
		return
	}

}

func listSegments(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	var segments []Segment
	err := db.Model(&segments).Select()
	if err != nil {
		http.Error(w, "Ошибка запроса к БД", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(segments)
	if err != nil {
		http.Error(w, "Json encode error", http.StatusInternalServerError)
		log.Fatal("Json encode error", err)
	}
}

func createSegment(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var newSegment Segment
	err := json.NewDecoder(r.Body).Decode(&newSegment)
	if err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	_, err = db.Model(&newSegment).Insert()
	if err != nil {
		http.Error(w, "Ошибка создания пользователя", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func updateSegment(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
	slug := routerParams.ByName("slug")

	var updatedSegment Segment
	err := json.NewDecoder(r.Body).Decode(&updatedSegment)
	if err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	updatedSegment.Name = slug
	res, err := db.Model(&updatedSegment).Where("name = ?", slug).Update()

	if err != nil {
		http.Error(w, "Ошибка обновления сегмента", http.StatusInternalServerError)
		return
	} else if res.RowsAffected() == 0 {
		http.Error(w, "Сегмент с таким slug отсутствует", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteSegment(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {

	slug := routerParams.ByName("slug")

	segment := &Segment{}
	res, err := db.Model(segment).Where("name = ?", slug).Delete()
	if err != nil {
		http.Error(w, "Ошибка удаления пользователя", http.StatusInternalServerError)
		return
	} else if res.RowsAffected() == 0 {
		http.Error(w, "Сегмент с таким slug отсутствует", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func listUsers(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	var users []User
	err := db.Model(&users).Select()
	if err != nil {
		http.Error(w, "Ошибка запроса к БД", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		http.Error(w, "Json encode error", http.StatusInternalServerError)
		log.Fatal("Json encode error", err)
	}
}

func createUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var newUser User
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	_, err = db.Model(&newUser).Insert()
	if err != nil {
		http.Error(w, "Ошибка создания пользователя", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getUser(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {

	userID := routerParams.ByName("id")

	user := &User{ID: -1}
	err := db.Model(user).Where("id = ?", userID).Select()
	if err != nil {
		http.Error(w, "Пользователь не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		http.Error(w, "Json encode error", http.StatusInternalServerError)
		log.Fatal("Json encode error", err)
	}
}

func updateUser(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
	userID := routerParams.ByName("id")

	var updatedUser User
	err := json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}

	updatedUser.ID, _ = strconv.Atoi(userID) // Convert string to int
	res, err := db.Model(&updatedUser).WherePK().Update()

	if err != nil {
		http.Error(w, "Ошибка обновления пользователя", http.StatusInternalServerError)
		return
	} else if res.RowsAffected() == 0 {
		http.Error(w, "Пользователь с таким id отсутствует", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteUser(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {

	userID := routerParams.ByName("id")

	user := &User{}
	res, err := db.Model(user).Where("id = ?", userID).Delete()
	if err != nil {
		http.Error(w, "Ошибка удаления пользователя", http.StatusInternalServerError)
		return
	} else if res.RowsAffected() == 0 {
		http.Error(w, "Пользователь с таким id отсутствует", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}