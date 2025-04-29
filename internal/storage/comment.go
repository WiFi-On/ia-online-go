package storage

import (
	"context"
	"errors"
	"fmt"
	"ia-online-golang/internal/models"
)

type CommentsRepositoryI interface {
	SaveComment(ctx context.Context, comment models.Comment) (models.Comment, error)
	Comments(ctx context.Context, leadID int64) ([]models.Comment, error)
}

var (
	ErrCommentsNotFound = errors.New("comments not found")
)

func (s *Storage) Comments(ctx context.Context, leadID int64) ([]models.Comment, error) {
	const op = "CommentRepository.Comments"

	query := "SELECT id, lead_id, user_id, text, created_at FROM comments WHERE lead_id = $1 ORDER BY created_at"

	rows, err := s.db.QueryContext(ctx, query, leadID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var comments []models.Comment

	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(
			&comment.ID,
			&comment.LeadID,
			&comment.UserID,
			&comment.Text,
			&comment.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(comments) == 0 {
		return nil, ErrCommentsNotFound
	}

	return comments, nil
}

func (s *Storage) SaveComment(ctx context.Context, comment models.Comment) (models.Comment, error) {
	const op = "CommentRepository.SaveComment"

	var (
		query string
		args  []interface{}
	)

	if comment.ID != 0 {
		query = `INSERT INTO comments (id, lead_id, user_id, text) VALUES ($1, $2, $3, $4)`
		args = []interface{}{comment.ID, comment.LeadID, comment.UserID, comment.Text}

		_, err := s.db.ExecContext(ctx, query, args...)
		if err != nil {
			return models.Comment{}, fmt.Errorf("%s: %w", op, err)
		}

		// При ручной установке ID можно отдельно получить created_at, если нужно
		query = `SELECT created_at FROM comments WHERE id = $1`
		err = s.db.QueryRowContext(ctx, query, comment.ID).Scan(&comment.CreatedAt)
		if err != nil {
			return models.Comment{}, fmt.Errorf("%s: %w", op, err)
		}

		return comment, nil
	}

	// Возвращаем и ID, и дату создания
	query = `INSERT INTO comments (lead_id, user_id, text) VALUES ($1, $2, $3) RETURNING id, created_at`
	args = []interface{}{comment.LeadID, comment.UserID, comment.Text}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt)
	if err != nil {
		return models.Comment{}, fmt.Errorf("%s: %w", op, err)
	}

	return comment, nil
}
