package repository

import (
	"context"
	"database/sql"
	"expense_tracker/domain"
	pkgrepo "expense_tracker/repository"
	"strconv"

	"github.com/google/uuid"
)

// CategoryRepoPG implements CategoryRepository with PostgreSQL
type CategoryRepoPG struct {
	db *sql.DB
}

// NewCategoryRepoPG returns a new PostgreSQL category repository
func NewCategoryRepoPG(db *sql.DB) *CategoryRepoPG {
	return &CategoryRepoPG{db: db}
}

func (r *CategoryRepoPG) Create(ctx context.Context, input domain.CreateCategoryInput) (*domain.Category, error) {
	id := uuid.New().String()
	var userID interface{}
	if input.UserID != nil {
		userID = *input.UserID
	} else {
		userID = nil
	}
	query := `INSERT INTO categories (id, name, user_id) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, id, input.Name, userID)
	if err != nil {
		return nil, err
	}
	return &domain.Category{
		ID:     id,
		Name:   input.Name,
		UserID: input.UserID,
	}, nil
}

func (r *CategoryRepoPG) GetByID(ctx context.Context, id string, userID *string) (*domain.Category, error) {
	// Category is visible if global (user_id IS NULL) or belongs to user
	query := `SELECT id, name, user_id FROM categories WHERE id = $1`
	args := []interface{}{id}
	if userID != nil {
		query += ` AND (user_id IS NULL OR user_id = $2)`
		args = append(args, *userID)
	}
	row := r.db.QueryRowContext(ctx, query, args...)
	var c domain.Category
	var uid sql.NullString
	err := row.Scan(&c.ID, &c.Name, &uid)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if uid.Valid {
		c.UserID = &uid.String
	}
	return &c, nil
}

// List returns categories: if userID is nil, only global; otherwise global + user's categories
func (r *CategoryRepoPG) List(ctx context.Context, userID *string, options pkgrepo.ListOptions) ([]*domain.Category, int, error) {
	var baseQuery string
	var args []interface{}
	if userID == nil {
		baseQuery = ` FROM categories WHERE user_id IS NULL`
		args = nil
	} else {
		baseQuery = ` FROM categories WHERE user_id IS NULL OR user_id = $1`
		args = []interface{}{*userID}
	}

	countQuery := `SELECT COUNT(*)` + baseQuery
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT id, name, user_id` + baseQuery + ` ORDER BY name LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, options.Limit, options.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []*domain.Category
	for rows.Next() {
		var c domain.Category
		var uid sql.NullString
		if err := rows.Scan(&c.ID, &c.Name, &uid); err != nil {
			return nil, 0, err
		}
		if uid.Valid {
			c.UserID = &uid.String
		}
		list = append(list, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *CategoryRepoPG) Update(ctx context.Context, id string, userID *string, input domain.UpdateCategoryInput) (*domain.Category, error) {
	existing, err := r.GetByID(ctx, id, userID)
	if err != nil || existing == nil {
		return nil, err
	}
	name := existing.Name
	if input.Name != nil {
		name = *input.Name
	}
	var query string
	var args []interface{}
	if userID != nil {
		// User can only update their own categories (not global)
		query = `UPDATE categories SET name = $1 WHERE id = $2 AND user_id = $3`
		args = []interface{}{name, id, *userID}
	} else {
		query = `UPDATE categories SET name = $1 WHERE id = $2 AND user_id IS NULL`
		args = []interface{}{name, id}
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	existing.Name = name
	return existing, nil
}

func (r *CategoryRepoPG) Delete(ctx context.Context, id string, userID *string) error {
	var query string
	var args []interface{}
	if userID == nil {
		query = `DELETE FROM categories WHERE id = $1 AND user_id IS NULL`
		args = []interface{}{id}
	} else {
		query = `DELETE FROM categories WHERE id = $1 AND user_id = $2`
		args = []interface{}{id, *userID}
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
