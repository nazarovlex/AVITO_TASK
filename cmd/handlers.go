package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/nazarovlex/AVITO_TASK/cmd/db"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type AddSegmentRequest struct {
	UserID          uuid.UUID      `json:"user_id"`
	SegmentsToAdd   map[string]int `json:"segments_to_add"`
	SegmentToDelete []string       `json:"segment_to_delete"`
}

type GetReportRequest struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

func getUsers(ctx context.Context, usersRegistry *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		users, err := usersRegistry.FetchUsers(ctx)
		if err != nil {
			http.Error(w, "fetch users error", http.StatusInternalServerError)
			log.Fatal("fetch users error ", err)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(users)
		if err != nil {
			http.Error(w, "Json encode error", http.StatusInternalServerError)
			log.Fatal("Json encode error", err)
		}
	}
}

func getUser(ctx context.Context, usersRegistry *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
		userId, err := uuid.Parse(routerParams.ByName("id"))
		if err != nil {
			http.Error(w, "UUID parse error", http.StatusInternalServerError)
			return
		}
		user, err := usersRegistry.FetchUser(ctx, userId)
		if err != nil {
			http.Error(w, "Users not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(user)
		if err != nil {
			http.Error(w, "Json encode error", http.StatusInternalServerError)
			log.Fatal("Json encode error", err)
		}
	}
}

func createUser(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var newUser db.Users
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			http.Error(w, "Invalid request data", http.StatusBadRequest)
			return
		}

		err = database.CreateUser(ctx, newUser.Name)
		if err != nil {
			http.Error(w, "Users creating error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}

}

func createSegment(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var newSegment db.Segments
		err := json.NewDecoder(r.Body).Decode(&newSegment)
		if err != nil {
			http.Error(w, "Invalid request data", http.StatusBadRequest)
			return
		}

		err = database.CreateSegment(ctx, newSegment.Slug)
		if err != nil {
			http.Error(w, "Segment creating error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func deleteSegment(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
		slug := routerParams.ByName("slug")

		err := database.DeleteSegment(ctx, slug)
		if err != nil {
			http.Error(w, "Deleting slug error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func updateSegment(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
		var updatedSegment db.Segments
		err := json.NewDecoder(r.Body).Decode(&updatedSegment)
		if err != nil {
			http.Error(w, "Invalid request data", http.StatusBadRequest)
			return
		}

		err = database.UpdateSegment(ctx, updatedSegment)
		if err != nil {
			http.Error(w, "Slug updating error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func addSegmentsToUser(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var requestData AddSegmentRequest
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "Invalid request data", http.StatusBadRequest)
			return
		}

		// check userId
		userExist := database.CheckExistedUser(ctx, requestData.UserID)
		if userExist == false {
			http.Error(w, "Users not found", http.StatusNotFound)
			return
		}

		currentTime := time.Now()
		var currentSegment db.Segments

		// delete segments
		for _, segment := range requestData.SegmentToDelete {
			currentSegment, err = database.FetchSegment(ctx, segment)
			if err != nil {
				http.Error(w, "Slug not found: "+segment, http.StatusBadRequest)
				return
			}
			err = database.DeleteUserSegments(ctx, requestData.UserID, currentSegment.ID)
			if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}

			operation := "удаление"
			err = database.SaveHistory(ctx, requestData.UserID, currentSegment.ID, operation, time.Now())
			if err != nil {
				http.Error(w, "History saving error", http.StatusInternalServerError)
				return
			}
		}

		// add new segments and expiration time to user
		for segment, ttl := range requestData.SegmentsToAdd {
			currentSegment, err = database.FetchSegment(ctx, segment)
			if err != nil {
				http.Error(w, "Slug not found: "+segment, http.StatusBadRequest)
				return
			}
			err = database.AddUserSegments(ctx, requestData.UserID, currentSegment.ID, currentTime.Add(time.Duration(ttl)*time.Hour))
			pgErr, ok := err.(pg.Error)
			if ok && pgErr.IntegrityViolation() {
				http.Error(w, "Some segment already added to user", http.StatusInternalServerError)
				return
			} else if err != nil {
				http.Error(w, "DB query error", http.StatusInternalServerError)
				return
			}
			operation := "добавление"
			err = database.SaveHistory(ctx, requestData.UserID, currentSegment.ID, operation, currentTime)
			if err != nil {
				http.Error(w, "History saving error", http.StatusInternalServerError)
				return
			}
		}

	}
}

func createReport(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var requestData GetReportRequest
		err := json.NewDecoder(r.Body).Decode(&requestData)
		if err != nil {
			http.Error(w, "Invalid request data", http.StatusBadRequest)
			return
		}

		entries, err := database.GetHistory(ctx, requestData.Year, requestData.Month)
		log.Println(entries)
		if err != nil {
			http.Error(w, "DB query error!!!!!!!!!!!!!!", http.StatusInternalServerError)
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
				"идентификатор пользователя " + entry.UserID.String(),
				entry.Slug,
				entry.Operation,
				entry.OperationAt.Format("2006-01-02 15:04:05"),
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

//func updateSegment(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
//	slug := routerParams.ByName("slug")
//
//	var updatedSegment Segment
//	err := json.NewDecoder(r.Body).Decode(&updatedSegment)
//	if err != nil {
//		http.Error(w, "Invalid request data", http.StatusBadRequest)
//		return
//	}
//
//	updatedSegment.Name = slug
//	res, err := db.Model(&updatedSegment).Where("name = ?", slug).Update()
//
//	if err != nil {
//		http.Error(w, "Slug updating error", http.StatusInternalServerError)
//		return
//	} else if res.RowsAffected() == 0 {
//		http.Error(w, "Slug not found", http.StatusNotFound)
//		return
//	}
//
//	w.WriteHeader(http.StatusOK)
//}

//

//func deleteUser(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
//	userID, _ := strconv.Atoi(routerParams.ByName("id"))
//
//	var user Users
//	err := db.Model(&user).Where("id = ?", userID).Select()
//	if err != nil {
//		http.Error(w, "Users not found", http.StatusNotFound)
//		return
//	}
//
//	var currentSegment Segment
//	for segmentName := range user.Segments {
//		err = db.Model(&currentSegment).Where("name=?", segmentName).Select()
//		if err != nil {
//			http.Error(w, "DB query error", http.StatusInternalServerError)
//			return
//		}
//
//		// delete user from segments table
//		delete(currentSegment.Users, userID)
//
//		// Update user in DB
//		_, err = db.Model(&currentSegment).Where("name=?", segmentName).Update()
//		if err != nil {
//			http.Error(w, "Slug db updating error", http.StatusInternalServerError)
//			return
//		}
//	}
//
//	_, err = db.Model(&user).Where("id = ?", userID).Delete()
//	if err != nil {
//		http.Error(w, "Users deleting error", http.StatusInternalServerError)
//		return
//	}
//
//	w.WriteHeader(http.StatusNoContent)
//}
//
//func getSegments(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
//	var segments []Segment
//	err := db.Model(&segments).Select()
//	if err != nil {
//		http.Error(w, "DB query error", http.StatusInternalServerError)
//		return
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	err = json.NewEncoder(w).Encode(segments)
//	if err != nil {
//		http.Error(w, "Json encode error", http.StatusInternalServerError)
//		log.Fatal("Json encode error", err)
//	}
//}
//
//func updateUser(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
//	userID := routerParams.ByName("id")
//
//	var updatedUser Users
//	err := json.NewDecoder(r.Body).Decode(&updatedUser)
//	if err != nil {
//		http.Error(w, "Invalid request data", http.StatusBadRequest)
//		return
//	}
//
//	updatedUser.ID, _ = strconv.Atoi(userID) // Convert string to int
//	res, err := db.Model(&updatedUser).WherePK().Update()
//
//	if err != nil {
//		http.Error(w, "Users updating error", http.StatusInternalServerError)
//		return
//	} else if res.RowsAffected() == 0 {
//		http.Error(w, "Users not found", http.StatusNotFound)
//		return
//	}
//
//	w.WriteHeader(http.StatusOK)
//}
