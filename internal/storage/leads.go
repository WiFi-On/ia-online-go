package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"ia-online-golang/internal/models"
	"strings"
	"time"
)

type LeadRepositoryI interface {
	LeadByID(ctx context.Context, id int64) (*models.Lead, error)
	CreateLead(ctx context.Context, lead *models.Lead) error
	Leads(ctx context.Context,
		statusID *int64,
		startDate, endDate *time.Time,
		limit, offset int64,
		userID *int64,
		Search *string,
		IsInternet, IsShipping, IsCleaning *bool) ([]models.Lead, error)
	UpdateLead(ctx context.Context,
		id, userID, statusID *int64,
		reward_internet, reward_cleaning, reward_shipping *float64,
		fio, phone_number, address *string,
		internet, cleaning, shipping *bool,
		created_at, completed_at, payment_at *time.Time) error
	DeleteLead(ctx context.Context, id int64) error
}

var (
	ErrLeadNotFound  = errors.New("lead not found")
	ErrLeadsNotFound = errors.New("leads not found")
)

func (s *Storage) LeadByID(ctx context.Context, id int64) (*models.Lead, error) {
	const op = "storage.leads.GetLeadByID"

	query := `
		SELECT id, user_id, fio, address, status_id, phone_number, internet, cleaning, shipping, created_at, completed_at, payment_at
		FROM leads
		WHERE id = $1
	`

	lead := &models.Lead{}
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&lead.ID, &lead.UserID, &lead.FIO, &lead.Address, &lead.StatusID, &lead.PhoneNumber, &lead.Internet,
		&lead.Cleaning, &lead.Shipping, &lead.CreatedAt, &lead.CompletedAt, &lead.PaymentAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrLeadNotFound
		}
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return lead, nil
}

func (s *Storage) CreateLead(ctx context.Context, lead *models.Lead) error {
	const op = "storage.leads.CreateLead"

	query := `
		INSERT INTO leads (id, user_id, fio, address, status_id, phone_number, internet, cleaning, shipping)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	// Выполнение запроса и возврат нового ID
	err := s.db.QueryRowContext(ctx, query,
		lead.ID, lead.UserID, lead.FIO, lead.Address, lead.StatusID, lead.PhoneNumber, lead.Internet,
		lead.Cleaning, lead.Shipping,
	).Scan(&lead.ID)

	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) Leads(ctx context.Context, statusID *int64, startDate, endDate *time.Time, limit, offset int64, userID *int64, Search *string, IsInternet, IsShipping, IsCleaning *bool) ([]models.Lead, error) {
	const op = "storage.leads.GetLeads"

	query := `
		SELECT id, user_id, fio, address, status_id, phone_number, internet, cleaning, shipping, created_at, completed_at, payment_at, reward_internet, reward_cleaning, reward_shipping
		FROM leads
		WHERE 1=1
	`

	var args []interface{}
	argCount := 1

	if statusID != nil {
		query += fmt.Sprintf(" AND status_id = $%d", argCount)
		args = append(args, *statusID)
		argCount++
	}

	if startDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *startDate)
		argCount++
	}

	if endDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *endDate)
		argCount++
	}

	if userID != nil {
		query += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *userID)
		argCount++
	}

	if IsInternet != nil {
		query += fmt.Sprintf(" AND internet = $%d", argCount)
		args = append(args, *IsInternet)
		argCount++
	}

	if IsShipping != nil {
		query += fmt.Sprintf(" AND shipping = $%d", argCount)
		args = append(args, *IsShipping)
		argCount++
	}

	if IsCleaning != nil {
		query += fmt.Sprintf(" AND cleaning = $%d", argCount)
		args = append(args, *IsCleaning)
		argCount++
	}

	if Search != nil && *Search != "" {
		query += fmt.Sprintf(`
			AND (
				LOWER(fio) ILIKE LOWER($%d) OR
				LOWER(address) ILIKE LOWER($%d) OR
				LOWER(phone_number) ILIKE LOWER($%d)
			)
		`, argCount, argCount, argCount)
		args = append(args, "%"+*Search+"%")
		argCount++
	}

	// Приоритетная сортировка по status_id (если задан), затем по created_at
	if statusID != nil {
		query += fmt.Sprintf(" ORDER BY CASE WHEN status_id = $%d THEN 0 ELSE 1 END, created_at DESC", argCount)
		args = append(args, *statusID)
		argCount++
	} else {
		query += " ORDER BY created_at DESC"
	}

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, limit)
		argCount++

		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, offset)
		argCount++
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var leads []models.Lead
	for rows.Next() {
		lead := models.Lead{}
		if err := rows.Scan(
			&lead.ID, &lead.UserID, &lead.FIO, &lead.Address, &lead.StatusID, &lead.PhoneNumber, &lead.Internet,
			&lead.Cleaning, &lead.Shipping, &lead.CreatedAt, &lead.CompletedAt, &lead.PaymentAt, &lead.RewardInternet, &lead.RewardCleaning, &lead.RewardShipping,
		); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}

		leads = append(leads, lead)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	if len(leads) == 0 {
		return nil, ErrLeadsNotFound
	}

	return leads, nil
}

func (s *Storage) UpdateLead(
	ctx context.Context,
	id, userID, statusID *int64,
	reward_internet, reward_cleaning, reward_shipping *float64,
	fio, phone_number, address *string,
	internet, cleaning, shipping *bool,
	created_at, completed_at, payment_at *time.Time,
) error {
	const op = "storage.leads.UpdateLead"

	query := "UPDATE leads SET"
	var args []interface{}
	argCount := 1
	var setClauses []string

	addClause := func(field string, value interface{}) {
		setClauses = append(setClauses, fmt.Sprintf(" %s = $%d", field, argCount))
		args = append(args, value)
		argCount++
	}

	if userID != nil {
		addClause("user_id", userID)
	}
	if statusID != nil {
		addClause("status_id", statusID)
	}
	if fio != nil {
		addClause("fio", fio)
	}
	if phone_number != nil {
		addClause("phone_number", phone_number)
	}
	if address != nil {
		addClause("address", address)
	}
	if internet != nil {
		addClause("internet", internet)
	}
	if cleaning != nil {
		addClause("cleaning", cleaning)
	}
	if shipping != nil {
		addClause("shipping", shipping)
	}
	if reward_internet != nil {
		addClause("reward_internet", reward_internet)
	}
	if reward_cleaning != nil {
		addClause("reward_cleaning", reward_cleaning)
	}
	if reward_shipping != nil {
		addClause("reward_shipping", reward_shipping)
	}
	if created_at != nil {
		addClause("created_at", created_at)
	}
	if completed_at != nil {
		addClause("completed_at", completed_at)
	}
	if payment_at != nil {
		addClause("payment_at", payment_at)
	}

	if len(setClauses) == 0 {
		// Нет данных для обновления
		return nil
	}

	query += strings.Join(setClauses, ",")
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, id)

	_, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) DeleteLead(ctx context.Context, id int64) error {
	const op = "storage.leads.DeleteLead"

	query := `DELETE FROM leads WHERE id = $1`

	// Выполнение запроса на удаление
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}
