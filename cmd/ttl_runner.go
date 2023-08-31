package main

import (
	"context"
	"github.com/nazarovlex/AVITO_TASK/cmd/db"
	"log"
	"time"
)

func runner(ctx context.Context, dbService *db.Service) {
	for {
		time.Sleep(1 * time.Hour)
		err := dbService.DropExpiredSegments(ctx)
		if err != nil {
			log.Printf("runner error %v\n", err)
		}
	}
}
