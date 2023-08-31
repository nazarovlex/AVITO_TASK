package db

import (
	"context"
	"github.com/google/uuid"
	"time"
)

// postgres models

type Users struct {
	tableName struct{}  `pg:"users"`
	ID        uuid.UUID `pg:"id,pk,type:uuid" json:"id"`
	Name      string    `pg:"name" json:"name"`
}

type SegmentAssignments struct {
	tableName struct{}  `pg:"segment_assignments"`
	UserID    uuid.UUID `pg:"user_id,type:uuid" json:"user_id"`
	SegmentID uuid.UUID `pg:"segment_id,type:uuid" json:"segment_id"`
	DeleteAt  time.Time `pg:"delete_at" json:"delete_at"`
}

type Segments struct {
	tableName struct{}  `pg:"segments"`
	ID        uuid.UUID `pg:"id,pk,type:uuid" json:"id"`
	Slug      string    `pg:"slug,unique" json:"slug" `
}

type UserSegmentHistory struct {
	tableName   struct{}  `pg:"user_segment_history"`
	UserID      uuid.UUID `pg:"user_id,type:uuid"`
	SegmentID   uuid.UUID `pg:"segment_id,type:uuid"`
	Operation   string    `pg:"operation,type:operation"`
	OperationAt time.Time `pg:"operation_at"`
}

// db response models

type GetHistory struct {
	UserID      uuid.UUID `pg:"user_id,type:uuid"`
	Operation   string    `pg:"operation"`
	OperationAt time.Time `pg:"operation_at"`
	Slug        string    `pg:"slug"`
}

type UserWithSegments struct {
	UserID       uuid.UUID `pg:"user_id,type:uuid"`
	SegmentSlugs []string  `pg:"segment_slugs,type:text[]"`
}

// create all models if not exist

func CreateSchema(ctx context.Context, dbService *Service) error {
	models := []interface{}{
		(*Users)(nil),
		(*Segments)(nil),
		(*SegmentAssignments)(nil),
		(*UserSegmentHistory)(nil),
	}

	for _, model := range models {
		err := dbService.CreateTable(ctx, model)
		if err != nil {
			return err
		}
	}
	return nil

}
