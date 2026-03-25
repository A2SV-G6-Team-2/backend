package repository

import (
	"context"
	"database/sql"
	"expense_tracker/domain"
	pkgrepo "expense_tracker/repository"
)

type DebtRepositoryPG struct {
	DB *sql.DB
}

func NewDebtRepositoryPG(db *sql.DB) *DebtRepositoryPG {
	return &DebtRepositoryPG{DB: db}
}

func (r *DebtRepositoryPG) Create(ctx context.Context, debt *domain.Debt) error {
	query := `
		INSERT INTO debts (
			id, user_id, type, peer_name, amount, due_date,
			reminder_enabled, remind_at, sent_at, status, note
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11
		)`

	_, err := r.DB.ExecContext(
		ctx,
		query,
		debt.ID,
		debt.UserID,
		debt.Type,
		debt.PeerName,
		debt.Amount,
		debt.DueDate,
		debt.ReminderEnabled,
		debt.RemindAt,
		debt.SentAt,
		debt.Status,
		debt.Note,
	)
	return err
}

func (r *DebtRepositoryPG) Update(ctx context.Context, debt *domain.Debt) error {
	query := `
		UPDATE debts
		SET type = $1,
			peer_name = $2,
			amount = $3,
			due_date = $4,
			reminder_enabled = $5,
			remind_at = $6,
			sent_at = $7,
			status = $8,
			note = $9
		WHERE id = $10
	`

	_, err := r.DB.ExecContext(
		ctx,
		query,
		debt.Type,
		debt.PeerName,
		debt.Amount,
		debt.DueDate,
		debt.ReminderEnabled,
		debt.RemindAt,
		debt.SentAt,
		debt.Status,
		debt.Note,
		debt.ID,
	)
	return err
}

func (r *DebtRepositoryPG) GetByID(ctx context.Context, id string) (*domain.Debt, error) {
	query := `
		SELECT id, user_id, type, peer_name, amount, due_date,
			reminder_enabled, remind_at, sent_at, status, note, created_at
		FROM debts
		WHERE id = $1
	`

	row := r.DB.QueryRowContext(ctx, query, id)
	return scanDebt(row)
}

func (r *DebtRepositoryPG) ListByUser(ctx context.Context, userID string, options pkgrepo.ListOptions) ([]*domain.Debt, int, error) {
	countQuery := `SELECT COUNT(*) FROM debts WHERE user_id = $1`
	var total int
	if err := r.DB.QueryRowContext(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, type, peer_name, amount, due_date,
			reminder_enabled, remind_at, sent_at, status, note, created_at
		FROM debts
		WHERE user_id = $1
		ORDER BY due_date ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.DB.QueryContext(ctx, query, userID, options.Limit, options.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanDebts(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *DebtRepositoryPG) ListUpcoming(ctx context.Context, userID string, days int, options pkgrepo.ListOptions) ([]*domain.Debt, int, error) {
	countQuery := `
		SELECT COUNT(*)
		FROM debts
		WHERE user_id = $1
			AND status = $2
			AND due_date >= CURRENT_DATE
			AND due_date <= CURRENT_DATE + ($3 * INTERVAL '1 day')
	`
	var total int
	if err := r.DB.QueryRowContext(ctx, countQuery, userID, domain.DebtStatusPending, days).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, type, peer_name, amount, due_date,
			reminder_enabled, remind_at, sent_at, status, note, created_at
		FROM debts
		WHERE user_id = $1
			AND status = $2
			AND due_date >= CURRENT_DATE
			AND due_date <= CURRENT_DATE + ($3 * INTERVAL '1 day')
		ORDER BY due_date ASC
		LIMIT $4 OFFSET $5
	`

	rows, err := r.DB.QueryContext(ctx, query, userID, domain.DebtStatusPending, days, options.Limit, options.Offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items, err := scanDebts(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *DebtRepositoryPG) MarkPaid(ctx context.Context, id string) (*domain.Debt, error) {
	query := `
		UPDATE debts
		SET status = $1,
			sent_at = NOW()
		WHERE id = $2
		RETURNING id, user_id, type, peer_name, amount, due_date,
			reminder_enabled, remind_at, sent_at, status, note, created_at
	`

	row := r.DB.QueryRowContext(ctx, query, domain.DebtStatusPaid, id)
	return scanDebt(row)
}

func (r *DebtRepositoryPG) SetOverdue(ctx context.Context, nowUTC string) (int64, error) {
	query := `
		UPDATE debts
		SET status = $1
		WHERE status = $2
			AND due_date < $3::date
	`

	result, err := r.DB.ExecContext(ctx, query, domain.DebtStatusOverdue, domain.DebtStatusPending, nowUTC)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (r *DebtRepositoryPG) GetDueForReminder(ctx context.Context, nowUTC string) ([]*domain.Debt, error) {
	query := `
		SELECT id, user_id, type, peer_name, amount, due_date,
			reminder_enabled, remind_at, sent_at, status, note, created_at
		FROM debts
		WHERE status = $1
			AND reminder_enabled = TRUE
			AND (
				due_date = $2::date
				OR due_date = ($2::date + INTERVAL '1 day')
				OR due_date = ($2::date + INTERVAL '3 day')
			)
			AND (sent_at IS NULL OR sent_at::date < $2::date)
		ORDER BY due_date ASC
	`

	rows, err := r.DB.QueryContext(ctx, query, domain.DebtStatusPending, nowUTC)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanDebts(rows)
}

func (r *DebtRepositoryPG) UpdateReminder(ctx context.Context, id string, remindAtUTC string, sentAtUTC string) error {
	query := `
		UPDATE debts
		SET remind_at = $1::timestamp,
			sent_at = $2::timestamp
		WHERE id = $3
	`

	_, err := r.DB.ExecContext(ctx, query, remindAtUTC, sentAtUTC, id)
	return err
}

type debtScanner interface {
	Scan(dest ...any) error
}

func scanDebt(row debtScanner) (*domain.Debt, error) {
	var debt domain.Debt
	var remindAt sql.NullTime
	var sentAt sql.NullTime
	var note sql.NullString

	if err := row.Scan(
		&debt.ID,
		&debt.UserID,
		&debt.Type,
		&debt.PeerName,
		&debt.Amount,
		&debt.DueDate,
		&debt.ReminderEnabled,
		&remindAt,
		&sentAt,
		&debt.Status,
		&note,
		&debt.CreatedAt,
	); err != nil {
		return nil, err
	}

	if remindAt.Valid {
		debt.RemindAt = &remindAt.Time
	}
	if sentAt.Valid {
		debt.SentAt = &sentAt.Time
	}
	if note.Valid {
		debt.Note = &note.String
	}

	return &debt, nil
}

func scanDebts(rows *sql.Rows) ([]*domain.Debt, error) {
	var debts []*domain.Debt
	for rows.Next() {
		debt, err := scanDebt(rows)
		if err != nil {
			return nil, err
		}
		debts = append(debts, debt)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return debts, nil
}
