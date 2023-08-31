package runner

import (
	"context"
	db2 "github.com/nazarovlex/AVITO_TASK/internal/db"
	"log"
	"time"
)

func Runner(ctx context.Context, dbService *db2.Service) {
	for {
		time.Sleep(1 * time.Hour)
		err := dbService.DropExpiredSegments(ctx)
		if err != nil {
			log.Printf("Runner error %v\n", err)
		}
	}
}
