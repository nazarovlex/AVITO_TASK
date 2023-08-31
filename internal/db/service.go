package db

import (
	"context"
	"github.com/google/uuid"
	"time"
)

type Service struct {
	db Database
}

func NewService(db Database) *Service {
	return &Service{
		db: db,
	}
}

type Database interface {
	//  postgres config
	CreateEnumType(ctx context.Context) error
	CreateTable(ctx context.Context, model interface{}) error
	CreateIndexes(ctx context.Context) error

	// user
	FetchUsers(ctx context.Context) ([]UserWithSegments, error)
	FetchUser(ctx context.Context, userId uuid.UUID) (UserWithSegments, error)
	CreateUser(ctx context.Context, name string) error
	DeleteUser(ctx context.Context, userId uuid.UUID) error

	// segments
	CreateSegment(ctx context.Context, slug string) error
	FetchSegment(ctx context.Context, slug string) (Segments, error)
	UpdateSegment(ctx context.Context, segment Segments) error
	DeleteSegment(ctx context.Context, slug string) error

	// adding and deleting user segments
	CheckExistedUser(ctx context.Context, userId uuid.UUID) bool
	AddUserSegments(ctx context.Context, userId, segmentId uuid.UUID, expirationTime time.Time) error
	DeleteUserSegments(ctx context.Context, userId, segmentId uuid.UUID) error

	// history
	SaveHistory(ctx context.Context, userId, segmentId uuid.UUID, operation string, operatedAt time.Time) error
	GetHistory(ctx context.Context, year, month int) ([]GetHistory, error)

	// runner
	DropExpiredSegments(ctx context.Context, timeNow time.Time) error
}

func (s *Service) CreateEnumType(ctx context.Context) error {
	err := s.db.CreateEnumType(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateTable(ctx context.Context, model interface{}) error {
	err := s.db.CreateTable(ctx, model)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateIndexes(ctx context.Context) error {
	err := s.db.CreateIndexes(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) FetchUsers(ctx context.Context) ([]UserWithSegments, error) {
	fetched, err := s.db.FetchUsers(ctx)
	if err != nil {
		return []UserWithSegments{}, err
	}
	return fetched, nil
}

func (s *Service) FetchUser(ctx context.Context, userId uuid.UUID) (UserWithSegments, error) {
	fetched, err := s.db.FetchUser(ctx, userId)
	if err != nil {
		return UserWithSegments{}, err
	}
	return fetched, nil
}

func (s *Service) CreateUser(ctx context.Context, name string) error {
	err := s.db.CreateUser(ctx, name)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteUser(ctx context.Context, userId uuid.UUID) error {
	err := s.db.DeleteUser(ctx, userId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CreateSegment(ctx context.Context, slug string) error {
	err := s.db.CreateSegment(ctx, slug)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) FetchSegment(ctx context.Context, slug string) (Segments, error) {
	fetched, err := s.db.FetchSegment(ctx, slug)
	if err != nil {
		return Segments{}, err
	}
	return fetched, nil
}

func (s *Service) UpdateSegment(ctx context.Context, segment Segments) error {
	err := s.db.UpdateSegment(ctx, segment)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteSegment(ctx context.Context, slug string) error {
	err := s.db.DeleteSegment(ctx, slug)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CheckExistedUser(ctx context.Context, userId uuid.UUID) bool {
	res := s.db.CheckExistedUser(ctx, userId)
	return res
}

func (s *Service) AddUserSegments(ctx context.Context, userId, segmentId uuid.UUID, expirationTime time.Time) error {
	err := s.db.AddUserSegments(ctx, userId, segmentId, expirationTime)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) DeleteUserSegments(ctx context.Context, userId, segmentId uuid.UUID) error {
	err := s.db.DeleteUserSegments(ctx, userId, segmentId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) SaveHistory(ctx context.Context, userId, segmentId uuid.UUID, operation string, operatedAt time.Time) error {
	err := s.db.SaveHistory(ctx, userId, segmentId, operation, operatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) GetHistory(ctx context.Context, year, month int) ([]GetHistory, error) {
	history, err := s.db.GetHistory(ctx, year, month)
	if err != nil {
		return []GetHistory{}, err
	}
	return history, nil
}

func (s *Service) DropExpiredSegments(ctx context.Context) error {
	timeNow := time.Now()
	err := s.db.DropExpiredSegments(ctx, timeNow)
	if err != nil {
		return err
	}
	return nil
}
