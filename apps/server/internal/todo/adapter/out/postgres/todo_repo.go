package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/todo/domain"
)

// TodoList repo

type listRepo struct {
	db *pgxpool.Pool
}

func NewListRepo(db *pgxpool.Pool) *listRepo {
	return &listRepo{db: db}
}

func (r *listRepo) Create(ctx context.Context, list *domain.TodoList) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		list.ID, list.UserID, list.GoogleID, list.GoogleAccountID, list.Name, list.SortOrder, list.CreatedAt, list.UpdatedAt,
	)
	return err
}

func (r *listRepo) FindByID(ctx context.Context, userID, id string) (*domain.TodoList, error) {
	var l domain.TodoList
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at
		 FROM todo_lists WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&l.ID, &l.UserID, &l.GoogleID, &l.GoogleAccountID, &l.Name, &l.SortOrder, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("list not found")
	}
	return &l, err
}

func (r *listRepo) ListByUser(ctx context.Context, userID string) ([]*domain.TodoList, error) {
	rows, err := r.db.Query(ctx,
		`SELECT l.id, l.user_id, l.google_id, l.google_account_id, uga.google_email, l.name, l.sort_order,
		        COALESCE(c.active_count, 0) AS active_count,
		        COALESCE(c.done_count, 0) AS done_count,
		        COALESCE(c.total_count, 0) AS total_count,
		        CASE
		          WHEN l.google_account_id IS NOT NULL AND l.google_account_id <> '' THEN 'google'
		          ELSE 'local'
		        END AS source,
		        l.created_at, l.updated_at
		   FROM todo_lists l
		   LEFT JOIN (
		     SELECT list_id,
		            COUNT(*) FILTER (WHERE deleted_at IS NULL AND is_done = FALSE) AS active_count,
		            COUNT(*) FILTER (WHERE deleted_at IS NULL AND is_done = TRUE) AS done_count,
		            COUNT(*) FILTER (WHERE deleted_at IS NULL) AS total_count
		       FROM todos
		      WHERE user_id = $1
		      GROUP BY list_id
		   ) c ON c.list_id = l.id
		   LEFT JOIN user_google_accounts uga
		          ON uga.id::text = l.google_account_id
		         AND uga.user_id::text = l.user_id
		  WHERE l.user_id = $1
		  ORDER BY l.sort_order, l.name`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTodoListRows(rows)
}

func (r *listRepo) Update(ctx context.Context, list *domain.TodoList) error {
	_, err := r.db.Exec(ctx,
		`UPDATE todo_lists SET name = $3, sort_order = $4, google_id = $5, google_account_id = $6, updated_at = $7
		 WHERE id = $1 AND user_id = $2`,
		list.ID, list.UserID, list.Name, list.SortOrder, list.GoogleID, list.GoogleAccountID, list.UpdatedAt,
	)
	return err
}

func (r *listRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM todo_lists WHERE id = $1`, id)
	return err
}

// Todo repo

type todoRepo struct {
	db *pgxpool.Pool
}

func NewTodoRepo(db *pgxpool.Pool) *todoRepo {
	return &todoRepo{db: db}
}

func (r *todoRepo) Create(ctx context.Context, todo *domain.Todo) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, parent_id, google_id, title, notes, due_date, due_time, priority, is_done, is_pinned, starred_at, sort_order, done_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		todo.ID, todo.ListID, todo.UserID, todo.ParentID, todo.GoogleID,
		todo.Title, todo.Notes, todo.DueDate, todo.DueTime, todo.Priority,
		todo.IsDone, todo.IsPinned, todo.StarredAt, todo.SortOrder, todo.DoneAt,
		todo.CreatedAt, todo.UpdatedAt,
	)
	return err
}

func (r *todoRepo) FindByID(ctx context.Context, userID, id string) (*domain.Todo, error) {
	var t domain.Todo
	err := r.db.QueryRow(ctx,
		`SELECT id, list_id, user_id, parent_id, google_id, title, notes,
		        CASE WHEN due_date IS NULL THEN NULL ELSE to_char(due_date, 'YYYY-MM-DD') END AS due_date,
		        CASE WHEN due_time IS NULL THEN NULL ELSE to_char(due_time, 'HH24:MI') END AS due_time,
		        priority, is_done, is_pinned, starred_at, sort_order, done_at, created_at, updated_at, deleted_at
		 FROM todos WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID,
	).Scan(&t.ID, &t.ListID, &t.UserID, &t.ParentID, &t.GoogleID,
		&t.Title, &t.Notes, &t.DueDate, &t.DueTime, &t.Priority,
		&t.IsDone, &t.IsPinned, &t.StarredAt, &t.SortOrder, &t.DoneAt,
		&t.CreatedAt, &t.UpdatedAt, &t.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("todo not found")
	}
	return &t, err
}

func (r *todoRepo) ListByList(ctx context.Context, userID, listID string, includeDone bool) ([]*domain.Todo, error) {
	query := `SELECT id, list_id, user_id, parent_id, google_id, title, notes,
		         CASE WHEN due_date IS NULL THEN NULL ELSE to_char(due_date, 'YYYY-MM-DD') END AS due_date,
		         CASE WHEN due_time IS NULL THEN NULL ELSE to_char(due_time, 'HH24:MI') END AS due_time,
		         priority, is_done, is_pinned, starred_at, sort_order, done_at, created_at, updated_at, deleted_at
		 FROM todos WHERE user_id = $1 AND list_id = $2 AND deleted_at IS NULL`

	if !includeDone {
		query += " AND is_done = FALSE"
	}

	query += " ORDER BY is_done ASC, sort_order ASC, created_at ASC"

	rows, err := r.db.Query(ctx, query, userID, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTodosRows(rows)
}

func (r *todoRepo) Update(ctx context.Context, todo *domain.Todo) error {
	_, err := r.db.Exec(ctx,
		`UPDATE todos SET list_id = $3, title = $4, notes = $5, due_date = $6, due_time = $7, priority = $8, is_done = $9, is_pinned = $10, starred_at = $11, sort_order = $12, done_at = $13, parent_id = $14, updated_at = $15
		 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		todo.ID, todo.UserID, todo.ListID, todo.Title, todo.Notes, todo.DueDate, todo.DueTime, todo.Priority,
		todo.IsDone, todo.IsPinned, todo.StarredAt, todo.SortOrder, todo.DoneAt, todo.ParentID, todo.UpdatedAt,
	)
	return err
}

func (r *todoRepo) SoftDelete(ctx context.Context, userID, id string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE todos SET deleted_at = $3, updated_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID, now,
	)
	return err
}

func (r *todoRepo) CountPinned(ctx context.Context, userID, listID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM todos WHERE user_id = $1 AND list_id = $2 AND is_pinned = TRUE AND deleted_at IS NULL`,
		userID, listID,
	).Scan(&count)
	return count, err
}

func (r *todoRepo) FindChildrenByParentID(ctx context.Context, userID, parentID string) ([]*domain.Todo, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, list_id, user_id, parent_id, google_id, title, notes,
		        CASE WHEN due_date IS NULL THEN NULL ELSE to_char(due_date, 'YYYY-MM-DD') END AS due_date,
		        CASE WHEN due_time IS NULL THEN NULL ELSE to_char(due_time, 'HH24:MI') END AS due_time,
		        priority, is_done, is_pinned, starred_at, sort_order, done_at, created_at, updated_at, deleted_at
		 FROM todos WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL
		 ORDER BY sort_order ASC`, userID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTodosRows(rows)
}

func (r *todoRepo) SoftDeleteByParentID(ctx context.Context, userID, parentID string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE todos SET deleted_at = $3, updated_at = $3 WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL`,
		userID, parentID, now,
	)
	return err
}

func (r *todoRepo) UpdateBatch(ctx context.Context, todos []*domain.Todo) error {
	batch := &pgx.Batch{}
	for _, t := range todos {
		batch.Queue(
			`UPDATE todos SET parent_id = $3, sort_order = $4, updated_at = $5 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
			t.ID, t.UserID, t.ParentID, t.SortOrder, t.UpdatedAt,
		)
	}
	br := r.db.SendBatch(ctx, batch)
	defer br.Close()

	for range todos {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

func (r *todoRepo) NextSortOrder(ctx context.Context, userID, listID string, parentID *string) (int, error) {
	var maxOrder int
	var err error
	if parentID == nil {
		err = r.db.QueryRow(ctx,
			`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM todos WHERE user_id = $1 AND list_id = $2 AND parent_id IS NULL AND deleted_at IS NULL`,
			userID, listID,
		).Scan(&maxOrder)
	} else {
		err = r.db.QueryRow(ctx,
			`SELECT COALESCE(MAX(sort_order), -1) + 1 FROM todos WHERE user_id = $1 AND list_id = $2 AND parent_id = $3 AND deleted_at IS NULL`,
			userID, listID, *parentID,
		).Scan(&maxOrder)
	}
	return maxOrder, err
}

func scanTodoListRows(rows pgx.Rows) ([]*domain.TodoList, error) {
	var lists []*domain.TodoList
	for rows.Next() {
		var l domain.TodoList
		if err := rows.Scan(
			&l.ID,
			&l.UserID,
			&l.GoogleID,
			&l.GoogleAccountID,
			&l.GoogleAccountEmail,
			&l.Name,
			&l.SortOrder,
			&l.ActiveCount,
			&l.DoneCount,
			&l.TotalCount,
			&l.Source,
			&l.CreatedAt,
			&l.UpdatedAt,
		); err != nil {
			return nil, err
		}
		lists = append(lists, &l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return lists, nil
}

func scanTodosRows(rows pgx.Rows) ([]*domain.Todo, error) {
	var todos []*domain.Todo
	for rows.Next() {
		var t domain.Todo
		if err := rows.Scan(&t.ID, &t.ListID, &t.UserID, &t.ParentID, &t.GoogleID,
			&t.Title, &t.Notes, &t.DueDate, &t.DueTime, &t.Priority,
			&t.IsDone, &t.IsPinned, &t.StarredAt, &t.SortOrder, &t.DoneAt,
			&t.CreatedAt, &t.UpdatedAt, &t.DeletedAt); err != nil {
			return nil, err
		}
		todos = append(todos, &t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return todos, nil
}
