package main

import (
	"encoding/json"
	"github.com/go-pg/pg/v10"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"strconv"
	"time"
)

func addSegmentsToUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var requestData AddSegmentRequest
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Неверные данные", http.StatusBadRequest)
		return
	}
	var currentSegment Segment
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

	// Creating var that storage wrong segments from request
	var notExistedSegments []string

	// Add new segments with TTL
	for segment, ttl := range requestData.SegmentsToAdd {
		err = db.Model(&Segment{}).Where("name=?", segment).Select()
		if err != nil {
			notExistedSegments = append(notExistedSegments, segment)
			continue
		}

		if _, ok := currentUser.Segments[segment]; (ok && requestData.Override) || !ok {
			current_ttl := time.Now().Add(time.Duration(ttl) * time.Hour)
			currentUser.Segments[segment] = current_ttl
			err = db.Model(&currentSegment).Where("name=?", segment).Select()
			if err != nil {
				http.Error(w, "Ошибка добавления пользователя в сегмент", http.StatusBadRequest)
				return
			}

			if currentSegment.Users == nil {
				currentSegment.Users = make(map[int]time.Time)
			}
			currentSegment.Users[currentUser.ID] = current_ttl
			_, err = db.Model(&currentSegment).Where("name=?", segment).Update()
			if err != nil {
				http.Error(w, "Ошибка добавления пользователя в сегмент", http.StatusBadRequest)
				return
			}
		}

	}

	// Update user in DB
	_, err = db.Model(&currentUser).WherePK().Update()
	if err != nil {
		http.Error(w, "Ошибка добавления сегментов пользователю", http.StatusBadRequest)
		return
	}

	// response
	if len(notExistedSegments) == 0 {
		err = json.NewEncoder(w).Encode("All segments added to user")
		if err != nil {
			http.Error(w, "Json encode error", http.StatusInternalServerError)
			log.Fatal("Json encode error", err)
		}
	} else {
		message := "Some slugs doesn't exist: "
		err = json.NewEncoder(w).Encode(map[string][]string{message: notExistedSegments})
		if err != nil {
			http.Error(w, "Json encode error", http.StatusInternalServerError)
			log.Fatal("Json encode error", err)
		}
	}

}

func getSegments(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
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
		pgErr, ok := err.(pg.Error)
		if ok && pgErr.IntegrityViolation() {
			http.Error(w, "Сегмент с таким именем уже существует", http.StatusInternalServerError)
			return
		} else {
			http.Error(w, "Ошибка создания сегмента", http.StatusInternalServerError)
			return
		}
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
	var segment Segment
	slug := routerParams.ByName("slug")

	err := db.Model(&segment).Where("name = ?", slug).Select()
	if err != nil {
		http.Error(w, "Сегмент с таким slug отсутствует", http.StatusInternalServerError)
		return
	}
	var currentUser User
	for userId := range segment.Users {
		err = db.Model(&currentUser).Where("id=?", userId).Select()
		if err != nil {
			http.Error(w, "Ошибка удаления сегмента", http.StatusInternalServerError)
			return
		}
		// delete segment from users table
		delete(currentUser.Segments, slug)

		// Update user in DB
		_, err = db.Model(&currentUser).WherePK().Update()
		if err != nil {
			http.Error(w, "Ошибка удаления сегмента у пользователя", http.StatusBadRequest)
			return
		}
	}

	_, err = db.Model(&segment).Where("name = ?", slug).Delete()
	if err != nil {
		http.Error(w, "Ошибка удаления сегмента", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getUsers(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
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
	userID, _ := strconv.Atoi(routerParams.ByName("id"))

	var user User
	err := db.Model(&user).Where("id = ?", userID).Select()
	if err != nil {
		http.Error(w, "Пользователь с таким id отсутствует", http.StatusInternalServerError)
		return
	}

	var currentSegment Segment
	for segmentName := range user.Segments {
		err = db.Model(&currentSegment).Where("name=?", segmentName).Select()
		if err != nil {
			http.Error(w, "Ошибка удаления сегмента", http.StatusInternalServerError)
			return
		}

		// delete user from segments table
		delete(currentSegment.Users, userID)

		// Update user in DB
		_, err = db.Model(&currentSegment).Where("name=?", segmentName).Update()
		if err != nil {
			http.Error(w, "Ошибка удаления пользователя у сегмента", http.StatusBadRequest)
			return
		}
	}

	_, err = db.Model(&user).Where("id = ?", userID).Delete()
	if err != nil {
		http.Error(w, "Ошибка удаления пользователя", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
