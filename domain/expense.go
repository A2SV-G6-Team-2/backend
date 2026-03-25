package domain

import "time"

// RecurrenceType for recurring expenses
type RecurrenceType string

const (
	RecurrenceDaily   RecurrenceType = "daily"
	RecurrenceWeekly  RecurrenceType = "weekly"
	RecurrenceMonthly RecurrenceType = "monthly"
)

// Expense represents a single expense record
type Expense struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Amount          float64        `json:"amount"`
	CategoryID      *string        `json:"category_id,omitempty"`
	IsRecurring     bool           `json:"is_recurring"`
	RecurrenceType  RecurrenceType `json:"recurrence_type,omitempty"`
	NextDueDate     *time.Time     `json:"next_due_date,omitempty"`
	ReminderEnabled bool           `json:"reminder_enabled"`
	ReminderSentAt  *time.Time     `json:"reminder_sent_at,omitempty"`
	Note            string         `json:"note,omitempty"`
	ExpenseDate     time.Time      `json:"expense_date"`
	CreatedAt       time.Time      `json:"created_at"`
}

// CreateExpenseInput is the input for creating an expense
type CreateExpenseInput struct {
	ID              string         `json:"id"`
	UserID          string         `json:"user_id"`
	Amount          float64        `json:"amount"`
	CategoryID      *string        `json:"category_id,omitempty"`
	IsRecurring     bool           `json:"is_recurring"`
	RecurrenceType  RecurrenceType `json:"recurrence_type,omitempty"`
	NextDueDate     *time.Time     `json:"next_due_date,omitempty"`
	ReminderEnabled bool           `json:"reminder_enabled"`
	Note            string         `json:"note,omitempty"`
	ExpenseDate     time.Time      `json:"expense_date"`
}

// UpdateExpenseInput is the input for updating an expense (partial update)
type UpdateExpenseInput struct {
	Amount          *float64        `json:"amount,omitempty"`
	CategoryID      *string         `json:"category_id,omitempty"`
	IsRecurring     *bool           `json:"is_recurring,omitempty"`
	RecurrenceType  *RecurrenceType `json:"recurrence_type,omitempty"`
	NextDueDate     *time.Time      `json:"next_due_date,omitempty"`
	ReminderEnabled *bool           `json:"reminder_enabled,omitempty"`
	Note            *string         `json:"note,omitempty"`
	ExpenseDate     *time.Time      `json:"expense_date,omitempty"`
}

// ExpenseFilter for listing expenses
type ExpenseFilter struct {
	UserID     string     // required for ownership
	CategoryID *string    // optional filter by category
	FromDate   *time.Time // optional start date (inclusive)
	ToDate     *time.Time // optional end date (inclusive)
	Limit      int
	Offset     int
}
