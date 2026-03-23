package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/gymbot/workout-tracker-back/internal/domain/entity"
	"github.com/gymbot/workout-tracker-back/internal/domain/repository"
	"github.com/gymbot/workout-tracker-back/pkg/apperror"
)

// DraftRoutineRepository implements repository.DraftRepository using PostgreSQL
type DraftRoutineRepository struct {
	conn *Connection
}

// Ensure DraftRoutineRepository implements the interface
var _ repository.DraftRepository = (*DraftRoutineRepository)(nil)

// NewDraftRoutineRepository creates a new DraftRoutineRepository
func NewDraftRoutineRepository(conn *Connection) *DraftRoutineRepository {
	return &DraftRoutineRepository{conn: conn}
}

// GetByCode retrieves a pending, non-expired draft by its short code
func (r *DraftRoutineRepository) GetByCode(ctx context.Context, code string) (*entity.DraftRoutine, error) {
	query := `
		SELECT draft_id, user_id, code, draft_data, status, created_at, expires_at
		FROM draft_routines
		WHERE code = $1
		  AND status = 'pending'
		  AND expires_at > NOW()
	`

	var draft entity.DraftRoutine
	var draftDataJSON []byte

	err := r.conn.DB.QueryRowContext(ctx, query, code).Scan(
		&draft.DraftID,
		&draft.UserID,
		&draft.Code,
		&draftDataJSON,
		&draft.Status,
		&draft.CreatedAt,
		&draft.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, apperror.NewNotFoundError("draft not found or expired")
	}
	if err != nil {
		return nil, apperror.NewInternalError("failed to query draft routine", err)
	}

	if err := json.Unmarshal(draftDataJSON, &draft.DraftData); err != nil {
		return nil, apperror.NewInternalError("failed to parse draft data", err)
	}

	return &draft, nil
}

// GetPhoneByUserID retrieves the user's full phone number
func (r *DraftRoutineRepository) GetPhoneByUserID(ctx context.Context, userID string) (string, error) {
	query := `SELECT full_phone_number FROM users WHERE user_id = $1`

	var phone string
	err := r.conn.DB.QueryRowContext(ctx, query, userID).Scan(&phone)
	if err == sql.ErrNoRows {
		return "", apperror.NewNotFoundError("user not found")
	}
	if err != nil {
		return "", apperror.NewInternalError("failed to query user phone", err)
	}

	return phone, nil
}

// UpdateDraftData persists the modified JSONB draft_data for a pending draft
func (r *DraftRoutineRepository) UpdateDraftData(ctx context.Context, code string, data entity.DraftData) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return apperror.NewInternalError("failed to marshal draft data", err)
	}

	query := `
		UPDATE draft_routines
		SET draft_data = $1
		WHERE code = $2
		  AND status = 'pending'
	`

	result, err := r.conn.DB.ExecContext(ctx, query, string(dataJSON), code)
	if err != nil {
		return apperror.NewInternalError("failed to update draft data", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperror.NewInternalError("failed to check rows affected", err)
	}

	if rowsAffected == 0 {
		return apperror.NewNotFoundError("draft not found or no longer pending")
	}

	return nil
}

// MarkApproved sets draft status to 'approved'
func (r *DraftRoutineRepository) MarkApproved(ctx context.Context, code string) error {
	query := `
		UPDATE draft_routines
		SET status = 'approved'
		WHERE code = $1
		  AND status = 'pending'
		  AND expires_at > NOW()
	`

	result, err := r.conn.DB.ExecContext(ctx, query, code)
	if err != nil {
		return apperror.NewInternalError("failed to approve draft", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperror.NewInternalError("failed to check rows affected", err)
	}

	if rowsAffected == 0 {
		return apperror.NewNotFoundError("draft not found, expired, or already approved")
	}

	return nil
}

// EnrichExercises fetches video_link and main_muscle for a batch of exercise IDs
func (r *DraftRoutineRepository) EnrichExercises(ctx context.Context, exerciseIDs []string) (map[string]repository.ExerciseMeta, error) {
	if len(exerciseIDs) == 0 {
		return map[string]repository.ExerciseMeta{}, nil
	}

	query := `
		SELECT exercise_id, COALESCE(link, ''), COALESCE(main_muscle, '')
		FROM exercises
		WHERE exercise_id = ANY($1::text[])
	`

	rows, err := r.conn.DB.QueryContext(ctx, query, exerciseIDs)
	if err != nil {
		return nil, apperror.NewInternalError("failed to query exercises for enrichment", err)
	}
	defer rows.Close()

	result := make(map[string]repository.ExerciseMeta, len(exerciseIDs))
	for rows.Next() {
		var id, link, muscle string
		if err := rows.Scan(&id, &link, &muscle); err != nil {
			fmt.Printf("[WARN] failed to scan exercise enrichment row: %v\n", err)
			continue
		}
		result[id] = repository.ExerciseMeta{
			VideoLink:  link,
			MainMuscle: muscle,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, apperror.NewInternalError("error iterating exercise enrichment rows", err)
	}

	return result, nil
}
