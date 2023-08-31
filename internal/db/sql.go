package db

import (
	"context"
	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"
	"log"
	"time"
)

type Sql struct {
	db *pg.DB
}

func NewSql(db *pg.DB) *Sql {
	return &Sql{
		db: db,
	}
}

func (s *Sql) CreateEnumType(ctx context.Context) error {
	query := `
    DO $$ BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'operation') THEN
            CREATE TYPE operation AS ENUM ('добавление', 'удаление');
        END IF;
    END $$;
`
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) CreateTable(ctx context.Context, model interface{}) error {
	err := s.db.ModelContext(ctx, model).CreateTable(&orm.CreateTableOptions{
		IfNotExists: true,
		Temp:        false,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) CreateIndexes(ctx context.Context) error {

	_, err := s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_user_id ON users (id)")
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_segment_id ON segments (id)")
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_segment_slug ON segments (slug)")
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS idx_user_segment ON segment_assignments(user_id, segment_id)")
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_delete_at ON segment_assignments (delete_at)")
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_user_id_history ON user_segment_history (user_id)")
	if err != nil {
		return err
	}

	return nil
}

func (s *Sql) DropExpiredSegments(ctx context.Context, timeNow time.Time) error {
	_, err := s.db.ModelContext(ctx, &SegmentAssignments{}).Where("delete_at < ?", timeNow).Delete()

	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) FetchUsers(ctx context.Context) ([]UserWithSegments, error) {
	var users []UserWithSegments
	query := `
    SELECT
    	u.id as user_id,
    	array_agg(sa_segments.slug) as segment_slugs
	FROM
    	users u
	LEFT JOIN (
    	SELECT
        	sa.user_id,
        	s.slug
    	FROM
        	segment_assignments sa
    	LEFT JOIN
        	segments s ON sa.segment_id = s.id
    	WHERE
        	s.id IS NOT NULL
) sa_segments ON u.id = sa_segments.user_id
	GROUP BY
    	u.id;

`

	_, err := s.db.QueryContext(ctx, &users, query)
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Sql) FetchUser(ctx context.Context, userId uuid.UUID) (UserWithSegments, error) {
	var user UserWithSegments
	query := `
	SELECT
		u.id as user_id,
		array_agg(sa_segments.slug) as segment_slugs
	FROM
		users u
	LEFT JOIN (
		SELECT
			sa.user_id,
			s.slug
		FROM
			segment_assignments sa
		LEFT JOIN
			segments s ON sa.segment_id = s.id
		WHERE
			s.id IS NOT NULL
	) sa_segments ON u.id = sa_segments.user_id
	WHERE u.id = ?
	GROUP BY
		u.id;
`
	_, err := s.db.QueryContext(ctx, &user, query, userId)
	if err != nil {
		return UserWithSegments{}, err
	}
	return user, nil
}

func (s *Sql) CreateUser(ctx context.Context, name string) error {
	user := Users{
		ID:   uuid.New(),
		Name: name,
	}
	_, err := s.db.ModelContext(ctx, &user).Insert()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) DeleteUser(ctx context.Context, userId uuid.UUID) error {
	_, err := s.db.ModelContext(ctx, &Users{}).Where("user_id=?", userId).Delete()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) CreateSegment(ctx context.Context, slug string) error {
	segment := Segments{
		ID:   uuid.New(),
		Slug: slug,
	}
	_, err := s.db.ModelContext(ctx, &segment).Insert()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) FetchSegment(ctx context.Context, slug string) (Segments, error) {
	var segment Segments
	err := s.db.ModelContext(ctx, &segment).Where("slug=?", slug).Select()
	if err != nil {
		return Segments{}, err
	}
	return segment, nil
}

func (s *Sql) UpdateSegment(ctx context.Context, segment Segments) error {
	_, err := s.db.ModelContext(ctx, &segment).Update()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) DeleteSegment(ctx context.Context, slug string) error {
	_, err := s.db.ModelContext(ctx, &Segments{}).Where("slug=?", slug).Delete()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) CheckExistedUser(ctx context.Context, userId uuid.UUID) bool {
	res, _ := s.db.ModelContext(ctx, &Users{}).Where("id=?", userId).Exists()
	return res
}

func (s *Sql) AddUserSegments(ctx context.Context, userId, segmentId uuid.UUID, expirationTime time.Time) error {
	segmentAssignment := SegmentAssignments{
		SegmentID: segmentId,
		UserID:    userId,
		DeleteAt:  expirationTime,
	}
	_, err := s.db.ModelContext(ctx, &segmentAssignment).Insert()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) DeleteUserSegments(ctx context.Context, userId, segmentId uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM segment_assignments WHERE user_id = ? AND segment_id = ?", userId, segmentId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) SaveHistory(ctx context.Context, userId, segmentId uuid.UUID, operation string, operatedAt time.Time) error {
	history := UserSegmentHistory{
		UserID:      userId,
		SegmentID:   segmentId,
		Operation:   operation,
		OperationAt: operatedAt,
	}
	_, err := s.db.ModelContext(ctx, &history).Insert()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sql) GetHistory(ctx context.Context, year, month int) ([]GetHistory, error) {
	var userSegmentsWithSlugs []GetHistory

	query := `
    SELECT
        user_segment_history.user_id,
        user_segment_history.operation,
        user_segment_history.operation_at,
        segments.slug
    FROM
        user_segment_history
    JOIN
        segments ON user_segment_history.segment_id = segments.id
    WHERE
        EXTRACT(year FROM user_segment_history.operation_at) = ?
    AND
        EXTRACT(month FROM user_segment_history.operation_at) = ?
`

	_, err := s.db.QueryContext(ctx, &userSegmentsWithSlugs, query, year, month)

	if err != nil {
		log.Println(err)
		return []GetHistory{}, err
	}
	return userSegmentsWithSlugs, nil
}
