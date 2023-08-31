package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/nazarovlex/AVITO_TASK/internal/db"
	"io"
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

func getUsers(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		users, err := database.FetchUsers(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("fetch users error: %v", err), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(users)
		if err != nil {
			http.Error(w, fmt.Sprintf("Json encode error: %v", err), http.StatusInternalServerError)
		}
	}
}

func getUser(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
		userId, err := uuid.Parse(routerParams.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("UUID parse error: %v", err), http.StatusInternalServerError)
			return
		}
		user, err := database.FetchUser(ctx, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("Users not found: %v", err), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(user)
		if err != nil {
			http.Error(w, fmt.Sprintf("Json encode error: %v", err), http.StatusInternalServerError)
		}
	}
}

func createUser(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var newUser db.Users
		err := json.NewDecoder(r.Body).Decode(&newUser)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request data: %v", err), http.StatusBadRequest)
			return
		}

		err = database.CreateUser(ctx, newUser.Name)
		if err != nil {
			http.Error(w, fmt.Sprintf("Users creating error: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func deleteUser(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
		userId, err := uuid.Parse(routerParams.ByName("id"))
		if err != nil {
			http.Error(w, fmt.Sprintf("UUID parse error: %v", err), http.StatusInternalServerError)
			return
		}

		err = database.DeleteUser(ctx, userId)
		if err != nil {
			http.Error(w, fmt.Sprintf("Users deleting error: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func createSegment(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var newSegment db.Segments
		err := json.NewDecoder(r.Body).Decode(&newSegment)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request data: %v", err), http.StatusBadRequest)
			return
		}

		err = database.CreateSegment(ctx, newSegment.Slug)
		if err != nil {
			http.Error(w, fmt.Sprintf("Segment creating error: %v", err), http.StatusInternalServerError)
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
			http.Error(w, fmt.Sprintf("Deleting slug error: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

func updateSegment(ctx context.Context, database *db.Service) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, routerParams httprouter.Params) {
		var updatedSegment db.Segments
		err := json.NewDecoder(r.Body).Decode(&updatedSegment)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid request data: %v", err), http.StatusBadRequest)
			return
		}

		err = database.UpdateSegment(ctx, updatedSegment)
		if err != nil {
			http.Error(w, fmt.Sprintf("Slug updating error: %v", err), http.StatusInternalServerError)
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
			http.Error(w, fmt.Sprintf("Invalid request data: %v", err), http.StatusBadRequest)
			return
		}

		// check userId
		userExist := database.CheckExistedUser(ctx, requestData.UserID)
		if userExist == false {
			http.Error(w, fmt.Sprintf("Users not found: %v", err), http.StatusNotFound)
			return
		}

		currentTime := time.Now()
		var currentSegment db.Segments

		// delete segments
		for _, segment := range requestData.SegmentToDelete {
			currentSegment, err = database.FetchSegment(ctx, segment)
			if err != nil {
				http.Error(w, fmt.Sprintf("Slug not found - %v error: %v", segment, err), http.StatusBadRequest)
				return
			}
			err = database.DeleteUserSegments(ctx, requestData.UserID, currentSegment.ID)
			if err != nil {
				http.Error(w, fmt.Sprintf("DB query error: %v", err), http.StatusInternalServerError)
				return
			}

			operation := "удаление"
			err = database.SaveHistory(ctx, requestData.UserID, currentSegment.ID, operation, time.Now())
			if err != nil {
				http.Error(w, fmt.Sprintf("History saving error: %v", err), http.StatusInternalServerError)
				return
			}
		}

		// add new segments and expiration time to user
		for segment, ttl := range requestData.SegmentsToAdd {
			currentSegment, err = database.FetchSegment(ctx, segment)
			if err != nil {
				http.Error(w, fmt.Sprintf("Slug not found - %v error: %v", segment, err), http.StatusBadRequest)
				return
			}
			err = database.AddUserSegments(ctx, requestData.UserID, currentSegment.ID, currentTime.Add(time.Duration(ttl)*time.Hour))
			pgErr, ok := err.(pg.Error)
			if ok && pgErr.IntegrityViolation() {
				http.Error(w, fmt.Sprintf("Some segment already added to user: %v", err), http.StatusInternalServerError)
				return
			} else if err != nil {
				http.Error(w, fmt.Sprintf("DB query error: %v", err), http.StatusInternalServerError)
				return
			}
			operation := "добавление"
			err = database.SaveHistory(ctx, requestData.UserID, currentSegment.ID, operation, currentTime)
			if err != nil {
				http.Error(w, fmt.Sprintf("History saving error: %v", err), http.StatusInternalServerError)
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
			http.Error(w, fmt.Sprintf("Invalid request data: %v", err), http.StatusBadRequest)
			return
		}

		entries, err := database.GetHistory(ctx, requestData.Year, requestData.Month)
		if err != nil {
			http.Error(w, fmt.Sprintf("DB query error: %v", err), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("report_%04d-%02d.csv", requestData.Year, requestData.Month)
		filepath := "reports/" + filename
		file, err := os.Create(filepath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Report creating error: %v", err), http.StatusInternalServerError)
			return
		}
		defer func(file *os.File) {
			err = file.Close()
			if err != nil {
				http.Error(w, fmt.Sprintf("Report creating error: %v", err), http.StatusInternalServerError)
				return
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
				http.Error(w, fmt.Sprintf("Report creating error: %v", err), http.StatusInternalServerError)
				return
			}
		}
		link := "localhost:8000/download_report/" + filename
		err = json.NewEncoder(w).Encode(link)
		if err != nil {
			http.Error(w, fmt.Sprintf("Json encode error: %v", err), http.StatusInternalServerError)
		}
	}
}

func downloadReport(w http.ResponseWriter, _ *http.Request, routerParams httprouter.Params) {
	filename := routerParams.ByName("filename")
	filePath := "reports/" + filename

	// Открываем файл для чтения
	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("File opening error: %v", err), http.StatusInternalServerError)
		return
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			http.Error(w, fmt.Sprintf("File closing error: %v", err), http.StatusInternalServerError)
			return
		}
	}(file)

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, fmt.Sprintf("File sending error: %v", err), http.StatusInternalServerError)
		return
	}
}
