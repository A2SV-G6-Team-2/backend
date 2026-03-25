package repository

import (
	"context"
	"database/sql"
	"expense_tracker/domain"
	pkgrepo "expense_tracker/repository"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// ExpenseRepoPG implements ExpenseRepository with PostgreSQL
type ExpenseRepoPG struct {
	db *sql.DB
}

// NewExpenseRepoPG returns a new PostgreSQL expense repository
func NewExpenseRepoPG(db *sql.DB) *ExpenseRepoPG {
	return &ExpenseRepoPG{db: db}
}

func (r *ExpenseRepoPG) Create(ctx context.Context, input domain.CreateExpenseInput) (*domain.Expense, error) {
	expenseID := input.ID
	if expenseID == "" {
		expenseID = uuid.New().String()
	}
	now := time.Now().UTC()

	var categoryID interface{}
	if input.CategoryID != nil {
		categoryID = *input.CategoryID
	} else {
		categoryID = nil
	}
	var nextDue interface{}
	if input.NextDueDate != nil {
		nextDue = input.NextDueDate.Format("2006-01-02")
	} else {
		nextDue = nil
	}

	query := `INSERT INTO expenses (
		id, user_id, amount, category_id, is_recurring, recurrence_type,
		next_due_date, reminder_enabled, note, expense_date, created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.db.ExecContext(ctx, query,
		expenseID, input.UserID, input.Amount, categoryID,
		input.IsRecurring, string(input.RecurrenceType), nextDue,
		input.ReminderEnabled, nullStr(input.Note), input.ExpenseDate.Format("2006-01-02"), now,
	)
	if err != nil {
		return nil, err
	}

	return &domain.Expense{
		ID:              expenseID,
		UserID:          input.UserID,
		Amount:          input.Amount,
		CategoryID:      input.CategoryID,
		IsRecurring:     input.IsRecurring,
		RecurrenceType:  input.RecurrenceType,
		NextDueDate:     input.NextDueDate,
		ReminderEnabled: input.ReminderEnabled,
		Note:            input.Note,
		ExpenseDate:     input.ExpenseDate,
		CreatedAt:       now,
	}, nil
}

func (r *ExpenseRepoPG) GetByID(ctx context.Context, id, userID string) (*domain.Expense, error) {
	query := `SELECT id, user_id, amount, category_id, is_recurring, recurrence_type,
		next_due_date, reminder_enabled, reminder_sent_at, note, expense_date, created_at
		FROM expenses WHERE id = $1 AND user_id = $2`
	row := r.db.QueryRowContext(ctx, query, id, userID)
	var e domain.Expense
	var catID sql.NullString
	var nextDue, expDate sql.NullTime
	var remSent sql.NullTime
	var note sql.NullString
	var recType sql.NullString
	err := row.Scan(
		&e.ID, &e.UserID, &e.Amount, &catID, &e.IsRecurring, &recType,
		&nextDue, &e.ReminderEnabled, &remSent, &note, &expDate, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if catID.Valid {
		e.CategoryID = &catID.String
	}
	if nextDue.Valid {
		e.NextDueDate = &nextDue.Time
	}
	e.ExpenseDate = expDate.Time
	if remSent.Valid {
		e.ReminderSentAt = &remSent.Time
	}
	if note.Valid {
		e.Note = note.String
	}
	if recType.Valid {
		e.RecurrenceType = domain.RecurrenceType(recType.String)
	}
	return &e, nil
}

func (r *ExpenseRepoPG) List(ctx context.Context, filter domain.ExpenseFilter) ([]*domain.Expense, int, error) {
	baseWhere := ` FROM expenses WHERE user_id = $1`
	args := []interface{}{filter.UserID}
	pos := 2
	if filter.CategoryID != nil {
		baseWhere += ` AND category_id = $` + strconv.Itoa(pos)
		args = append(args, *filter.CategoryID)
		pos++
	}
	if filter.FromDate != nil {
		baseWhere += ` AND expense_date >= $` + strconv.Itoa(pos)
		args = append(args, filter.FromDate.Format("2006-01-02"))
		pos++
	}
	if filter.ToDate != nil {
		baseWhere += ` AND expense_date <= $` + strconv.Itoa(pos)
		args = append(args, filter.ToDate.Format("2006-01-02"))
		pos++
	}

	countQuery := `SELECT COUNT(*)` + baseWhere
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, user_id, amount, category_id, is_recurring, recurrence_type,
		next_due_date, reminder_enabled, reminder_sent_at, note, expense_date, created_at` +
		baseWhere +
		` ORDER BY expense_date DESC, created_at DESC LIMIT $` + strconv.Itoa(pos) +
		` OFFSET $` + strconv.Itoa(pos+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items, err := scanExpenses(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *ExpenseRepoPG) Update(ctx context.Context, id, userID string, input domain.UpdateExpenseInput) (*domain.Expense, error) {
	// Fetch existing for ownership and to merge
	existing, err := r.GetByID(ctx, id, userID)
	if err != nil || existing == nil {
		return nil, err
	}

	amount := existing.Amount
	if input.Amount != nil {
		amount = *input.Amount
	}
	catID := existing.CategoryID
	if input.CategoryID != nil {
		catID = input.CategoryID
	}
	isRecurring := existing.IsRecurring
	if input.IsRecurring != nil {
		isRecurring = *input.IsRecurring
	}
	recType := existing.RecurrenceType
	if input.RecurrenceType != nil {
		recType = *input.RecurrenceType
	}
	nextDue := existing.NextDueDate
	if input.NextDueDate != nil {
		nextDue = input.NextDueDate
	}
	remEnabled := existing.ReminderEnabled
	if input.ReminderEnabled != nil {
		remEnabled = *input.ReminderEnabled
	}
	note := existing.Note
	if input.Note != nil {
		note = *input.Note
	}
	expDate := existing.ExpenseDate
	if input.ExpenseDate != nil {
		expDate = *input.ExpenseDate
	}

	var categoryID interface{}
	if catID != nil {
		categoryID = *catID
	} else {
		categoryID = nil
	}
	var nextDueVal interface{}
	if nextDue != nil {
		nextDueVal = nextDue.Format("2006-01-02")
	} else {
		nextDueVal = nil
	}

	query := `UPDATE expenses SET
		amount = $1, category_id = $2, is_recurring = $3, recurrence_type = $4,
		next_due_date = $5, reminder_enabled = $6, note = $7, expense_date = $8
		WHERE id = $9 AND user_id = $10`
	_, err = r.db.ExecContext(ctx, query,
		amount, categoryID, isRecurring, string(recType), nextDueVal,
		remEnabled, nullStr(note), expDate.Format("2006-01-02"), id, userID,
	)
	if err != nil {
		return nil, err
	}
	existing.Amount = amount
	existing.CategoryID = catID
	existing.IsRecurring = isRecurring
	existing.RecurrenceType = recType
	existing.NextDueDate = nextDue
	existing.ReminderEnabled = remEnabled
	existing.Note = note
	existing.ExpenseDate = expDate
	return existing, nil
}

func (r *ExpenseRepoPG) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM expenses WHERE id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// SumByDateRange returns total expense amount for the user in the date range (report usecase)
func (r *ExpenseRepoPG) SumByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) (float64, error) {
	query := `SELECT COALESCE(SUM(amount), 0) FROM expenses WHERE user_id = $1 AND expense_date >= $2 AND expense_date <= $3`
	var total sql.NullFloat64
	if err := r.db.QueryRowContext(ctx, query, userID.String(), startDate, endDate).Scan(&total); err != nil {
		return 0, err
	}
	if !total.Valid {
		return 0, nil
	}
	return total.Float64, nil
}

// CategoryBreakdownByDateRange returns per-category totals for the user in the date range (report usecase)
func (r *ExpenseRepoPG) CategoryBreakdownByDateRange(ctx context.Context, userID uuid.UUID, startDate, endDate time.Time) ([]pkgrepo.CategoryTotal, error) {
	query := `SELECT COALESCE(c.name, 'Uncategorized') AS category_name, COALESCE(SUM(e.amount), 0) AS total
		FROM expenses e LEFT JOIN categories c ON e.category_id = c.id
		WHERE e.user_id = $1 AND e.expense_date >= $2 AND e.expense_date <= $3
		GROUP BY category_name ORDER BY total DESC`
	rows, err := r.db.QueryContext(ctx, query, userID.String(), startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []pkgrepo.CategoryTotal
	for rows.Next() {
		var name string
		var total sql.NullFloat64
		if err := rows.Scan(&name, &total); err != nil {
			return nil, err
		}
		t := 0.0
		if total.Valid {
			t = total.Float64
		}
		results = append(results, pkgrepo.CategoryTotal{CategoryName: name, Total: t})
	}
	return results, rows.Err()
}

func scanExpenses(rows *sql.Rows) ([]*domain.Expense, error) {
	var list []*domain.Expense
	for rows.Next() {
		var e domain.Expense
		var catID sql.NullString
		var nextDue, expDate sql.NullTime
		var remSent sql.NullTime
		var note sql.NullString
		var recType sql.NullString
		err := rows.Scan(
			&e.ID, &e.UserID, &e.Amount, &catID, &e.IsRecurring, &recType,
			&nextDue, &e.ReminderEnabled, &remSent, &note, &expDate, &e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if catID.Valid {
			e.CategoryID = &catID.String
		}
		if nextDue.Valid {
			e.NextDueDate = &nextDue.Time
		}
		e.ExpenseDate = expDate.Time
		if remSent.Valid {
			e.ReminderSentAt = &remSent.Time
		}
		if note.Valid {
			e.Note = note.String
		}
		if recType.Valid {
			e.RecurrenceType = domain.RecurrenceType(recType.String)
		}
		list = append(list, &e)
	}
	return list, rows.Err()
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
