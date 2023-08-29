package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/julienschmidt/httprouter"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

func addSegmentsToUser(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var requestData AddSegmentRequest
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	var currentSegment Segment
	var currentUser User
	err = db.Model(&currentUser).Where("id=?", requestData.UserID).Select()
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// map init
	if currentUser.Segments == nil {
		currentUser.Segments = make(map[string]time.Time)
	}

	// Delete segments
	for _, segment := range requestData.SegmentToDelete {
		delete(currentUser.Segments, segment)
		history := UserSegmentHistory{
			UserID:    requestData.UserID,
			Slug:      segment,
			Operation: "удаление",
			Timestamp: time.Now(),
		}
		_, err = db.Model(&history).Insert()
		if err != nil {
			http.Error(w, "Error history saving", http.StatusInternalServerError)
			return
		}
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
			expirationTime := time.Now().Add(time.Duration(ttl) * time.Hour)
			currentUser.Segments[segment] = expirationTime
			err = db.Model(&currentSegment).Where("name=?", segment).Select()
			if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}

			if currentSegment.Users == nil {
				currentSegment.Users = make(map[int]time.Time)
			}
			currentSegment.Users[currentUser.ID] = expirationTime
			_, err = db.Model(&currentSegment).Where("name=?", segment).Update()
			if err != nil {
				http.Error(w, "Error adding user to slug.", http.StatusInternalServerError)
				return
			}

			history := UserSegmentHistory{
				UserID:    requestData.UserID,
				Slug:      segment,
				Operation: "добавление",
				Timestamp: time.Now(),
			}
			_, err = db.Model(&history).Insert()
			if err != nil {
				http.Error(w, "Error history saving", http.StatusInternalServerError)
				return
			}

		}

	}

	// Update user in DB
	_, err = db.Model(&currentUser).WherePK().Update()
	if err != nil {
		http.Error(w, "Error adding slugs to user.", http.StatusInternalServerError)
		return
	}

	// response
	if len(notExistedSegments) == 0 {
		err = json.NewEncoder(w).Encode("All slugs added to user")
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
		http.Error(w, "DB query error", http.StatusInternalServerError)
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
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	// check auto adding user percent
	if newSegment.UsersPercent > 0 {
		var userCount int
		userCount, err = db.Model(&User{}).Count()
		if err != nil {
			http.Error(w, "Slug creating error", http.StatusInternalServerError)
			return
		}

		userToAdd := int(float64(userCount) / 100.00 * float64(newSegment.UsersPercent))

		var users []User
		err = db.Model(&users).OrderExpr("RANDOM()").Limit(userToAdd).Select()
		if err != nil {
			http.Error(w, "DB query error", http.StatusInternalServerError)
			return
		}

		expirationTime := time.Now().Add(time.Duration(newSegment.TTL) * time.Hour)

		for _, user := range users {
			if user.Segments == nil {
				user.Segments = make(map[string]time.Time)
			}
			if newSegment.Users == nil {
				newSegment.Users = make(map[int]time.Time)
			}
			user.Segments[newSegment.Name] = expirationTime
			_, err = db.Model(&user).WherePK().Update()
			if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}
			newSegment.Users[user.ID] = expirationTime
		}

	}

	_, err = db.Model(&newSegment).Insert()
	if err != nil {
		pgErr, ok := err.(pg.Error)
		if ok && pgErr.IntegrityViolation() {
			http.Error(w, "Slug with this name already exist", http.StatusConflict)
			return
		} else {
			http.Error(w, "Slug creating error", http.StatusInternalServerError)
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
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	updatedSegment.Name = slug
	res, err := db.Model(&updatedSegment).Where("name = ?", slug).Update()

	if err != nil {
		http.Error(w, "Slug updating error", http.StatusInternalServerError)
		return
	} else if res.RowsAffected() == 0 {
		http.Error(w, "Slug not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteSegment(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
	var segment Segment
	slug := routerParams.ByName("slug")

	err := db.Model(&segment).Where("name = ?", slug).Select()
	if err != nil {
		http.Error(w, "Slug not exist", http.StatusNotFound)
		return
	}
	var currentUser User
	for userId := range segment.Users {
		err = db.Model(&currentUser).Where("id=?", userId).Select()
		if err != nil {
			http.Error(w, "DB query error", http.StatusInternalServerError)
			return
		}
		// delete segment from users table
		delete(currentUser.Segments, slug)

		// Update user in DB
		_, err = db.Model(&currentUser).WherePK().Update()
		if err != nil {
			http.Error(w, "Deleting users slug error", http.StatusInternalServerError)
			return
		}
	}

	_, err = db.Model(&segment).Where("name = ?", slug).Delete()
	if err != nil {
		http.Error(w, "Deleting slug error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getUsers(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	var users []User
	err := db.Model(&users).Select()
	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
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
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	_, err = db.Model(&newUser).Insert()
	if err != nil {
		http.Error(w, "User creating error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func getUser(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {

	userID := routerParams.ByName("id")

	user := &User{ID: -1}
	err := db.Model(user).Where("id = ?", userID).Select()
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
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
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}

	updatedUser.ID, _ = strconv.Atoi(userID) // Convert string to int
	res, err := db.Model(&updatedUser).WherePK().Update()

	if err != nil {
		http.Error(w, "User updating error", http.StatusInternalServerError)
		return
	} else if res.RowsAffected() == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteUser(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
	userID, _ := strconv.Atoi(routerParams.ByName("id"))

	var user User
	err := db.Model(&user).Where("id = ?", userID).Select()
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var currentSegment Segment
	for segmentName := range user.Segments {
		err = db.Model(&currentSegment).Where("name=?", segmentName).Select()
		if err != nil {
			http.Error(w, "DB query error", http.StatusInternalServerError)
			return
		}

		// delete user from segments table
		delete(currentSegment.Users, userID)

		// Update user in DB
		_, err = db.Model(&currentSegment).Where("name=?", segmentName).Update()
		if err != nil {
			http.Error(w, "Slug users updating error", http.StatusInternalServerError)
			return
		}
	}

	_, err = db.Model(&user).Where("id = ?", userID).Delete()
	if err != nil {
		http.Error(w, "User deleting error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createReport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var requestData GetReportRequest
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		http.Error(w, "Invalid request data", http.StatusBadRequest)
		return
	}
	var entries []UserSegmentHistory
	err = db.Model(&entries).
		Where("EXTRACT(year FROM timestamp) = ?", requestData.Year).
		Where("EXTRACT(month FROM timestamp) = ?", requestData.Month).
		Select()

	if err != nil {
		http.Error(w, "DB query error", http.StatusInternalServerError)
		return
	}
	filename := fmt.Sprintf("report_%04d-%02d.csv", requestData.Year, requestData.Month)
	filepath := "reports/" + filename
	file, err := os.Create(filepath)
	if err != nil {
		http.Error(w, "Report creating error", http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {

		}
	}(file)

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, entry := range entries {
		err = writer.Write([]string{
			"идентификатор пользователя " + strconv.Itoa(entry.UserID),
			entry.Slug,
			entry.Operation,
			entry.Timestamp.Format("2006-01-02 15:04:05"),
		})
		if err != nil {
			return
		}
	}
	URL := "localhost:8000"
	link := URL + "/download_report/" + filename
	err = json.NewEncoder(w).Encode(map[string]string{"download_link": link})
	if err != nil {
		http.Error(w, "Json encode error", http.StatusInternalServerError)
		log.Fatal("Json encode error", err)
	}
}

func downloadReport(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
	filename := routerParams.ByName("filename")
	filePath := "reports/" + filename

	// Открываем файл для чтения
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File opening error", http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {

		}
	}(file)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "File sending error", http.StatusInternalServerError)
		return
	}
}
