package target

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	mu sync.Mutex
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directories: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Create table query
	query := `
	CREATE TABLE IF NOT EXISTS targets (
		id TEXT PRIMARY KEY,
		bot_name TEXT,
		symbol TEXT,
		target_price REAL,
		direction TEXT,
		created_at DATETIME,
		last_state TEXT,
		type TEXT,
		trailing_percent REAL,
		trailing_value REAL,
		activation_price REAL,
		is_active INTEGER,
		extreme_price REAL,
		note TEXT
	);`

	if _, err := db.Exec(query); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create targets table: %w", err)
	}

	return &SQLiteStore{
		db: db,
	}, nil
}

func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Close()
}

func (s *SQLiteStore) Save(target Target) (Target, error) {
	if target.Symbol == "" {
		return Target{}, fmt.Errorf("symbol is required")
	}

	target.Symbol = strings.ToUpper(target.Symbol)
	target.Direction = Direction(strings.ToLower(string(target.Direction)))
	if !IsValidDirection(string(target.Direction)) {
		return Target{}, fmt.Errorf("invalid direction: must be 'up' or 'down'")
	}

	if target.Type == "" {
		target.Type = TargetTypePrice
	}

	if target.Type == TargetTypePrice {
		if target.TargetPrice <= 0 {
			return Target{}, fmt.Errorf("target_price must be greater than zero for price alerts")
		}
	} else if target.Type == TargetTypeTrailing {
		if target.TrailingPercent <= 0 && target.TrailingValue <= 0 {
			return Target{}, fmt.Errorf("either trailing_percent or trailing_value must be greater than zero for trailing stop alerts")
		}
		if target.ActivationPrice <= 0 {
			target.IsActive = true
		} else {
			target.IsActive = false
		}
	} else {
		return Target{}, fmt.Errorf("invalid target type: %s", target.Type)
	}

	target.ID = generateID(target.Symbol)
	target.CreatedAt = time.Now().UTC()
	target.LastState = StateUnknown

	s.mu.Lock()
	defer s.mu.Unlock()

	isActiveInt := 0
	if target.IsActive {
		isActiveInt = 1
	}

	query := `INSERT INTO targets (
		id, bot_name, symbol, target_price, direction, created_at, last_state, 
		type, trailing_percent, trailing_value, activation_price, is_active, extreme_price, note
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query,
		target.ID, target.BotName, target.Symbol, target.TargetPrice, string(target.Direction),
		target.CreatedAt, string(target.LastState), string(target.Type), target.TrailingPercent,
		target.TrailingValue, target.ActivationPrice, isActiveInt, target.ExtremePrice, target.Note,
	)
	if err != nil {
		return Target{}, fmt.Errorf("failed to insert target: %w", err)
	}

	return target, nil
}

func (s *SQLiteStore) Get(id string) (Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `SELECT 
		id, bot_name, symbol, target_price, direction, created_at, last_state, 
		type, trailing_percent, trailing_value, activation_price, is_active, extreme_price, note
	FROM targets WHERE id = ?`

	var t Target
	var dir, lastState, tType string
	var isActiveInt int

	err := s.db.QueryRow(query, id).Scan(
		&t.ID, &t.BotName, &t.Symbol, &t.TargetPrice, &dir, &t.CreatedAt, &lastState,
		&tType, &t.TrailingPercent, &t.TrailingValue, &t.ActivationPrice, &isActiveInt, &t.ExtremePrice, &t.Note,
	)
	if err == sql.ErrNoRows {
		return Target{}, fmt.Errorf("target not found")
	} else if err != nil {
		return Target{}, fmt.Errorf("failed to query target: %w", err)
	}

	t.Direction = Direction(dir)
	t.LastState = State(lastState)
	t.Type = TargetType(tType)
	t.IsActive = isActiveInt == 1

	return t, nil
}

func (s *SQLiteStore) ListAll() ([]Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `SELECT 
		id, bot_name, symbol, target_price, direction, created_at, last_state, 
		type, trailing_percent, trailing_value, activation_price, is_active, extreme_price, note
	FROM targets`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all targets: %w", err)
	}
	defer rows.Close()

	var list []Target
	for rows.Next() {
		var t Target
		var dir, lastState, tType string
		var isActiveInt int

		err := rows.Scan(
			&t.ID, &t.BotName, &t.Symbol, &t.TargetPrice, &dir, &t.CreatedAt, &lastState,
			&tType, &t.TrailingPercent, &t.TrailingValue, &t.ActivationPrice, &isActiveInt, &t.ExtremePrice, &t.Note,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan target row: %w", err)
		}
		t.Direction = Direction(dir)
		t.LastState = State(lastState)
		t.Type = TargetType(tType)
		t.IsActive = isActiveInt == 1
		list = append(list, t)
	}

	return list, nil
}

func (s *SQLiteStore) ListBySymbol(symbol string) ([]Target, error) {
	symbol = strings.ToUpper(symbol)
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `SELECT 
		id, bot_name, symbol, target_price, direction, created_at, last_state, 
		type, trailing_percent, trailing_value, activation_price, is_active, extreme_price, note
	FROM targets WHERE UPPER(symbol) = ?`

	rows, err := s.db.Query(query, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to query targets by symbol: %w", err)
	}
	defer rows.Close()

	var list []Target
	for rows.Next() {
		var t Target
		var dir, lastState, tType string
		var isActiveInt int

		err := rows.Scan(
			&t.ID, &t.BotName, &t.Symbol, &t.TargetPrice, &dir, &t.CreatedAt, &lastState,
			&tType, &t.TrailingPercent, &t.TrailingValue, &t.ActivationPrice, &isActiveInt, &t.ExtremePrice, &t.Note,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan target row: %w", err)
		}
		t.Direction = Direction(dir)
		t.LastState = State(lastState)
		t.Type = TargetType(tType)
		t.IsActive = isActiveInt == 1
		list = append(list, t)
	}

	return list, nil
}

func (s *SQLiteStore) ListByBotName(botName string) ([]Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `SELECT 
		id, bot_name, symbol, target_price, direction, created_at, last_state, 
		type, trailing_percent, trailing_value, activation_price, is_active, extreme_price, note
	FROM targets WHERE bot_name = ?`

	rows, err := s.db.Query(query, botName)
	if err != nil {
		return nil, fmt.Errorf("failed to query targets by bot name: %w", err)
	}
	defer rows.Close()

	var list []Target
	for rows.Next() {
		var t Target
		var dir, lastState, tType string
		var isActiveInt int

		err := rows.Scan(
			&t.ID, &t.BotName, &t.Symbol, &t.TargetPrice, &dir, &t.CreatedAt, &lastState,
			&tType, &t.TrailingPercent, &t.TrailingValue, &t.ActivationPrice, &isActiveInt, &t.ExtremePrice, &t.Note,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan target row: %w", err)
		}
		t.Direction = Direction(dir)
		t.LastState = State(lastState)
		t.Type = TargetType(tType)
		t.IsActive = isActiveInt == 1
		list = append(list, t)
	}

	return list, nil
}

func (s *SQLiteStore) Update(target Target) error {
	if target.ID == "" {
		return fmt.Errorf("target id is required")
	}

	target.Symbol = strings.ToUpper(target.Symbol)
	isActiveInt := 0
	if target.IsActive {
		isActiveInt = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE targets SET 
		bot_name = ?, symbol = ?, target_price = ?, direction = ?, created_at = ?, last_state = ?, 
		type = ?, trailing_percent = ?, trailing_value = ?, activation_price = ?, is_active = ?, extreme_price = ?, note = ?
	WHERE id = ?`

	res, err := s.db.Exec(query,
		target.BotName, target.Symbol, target.TargetPrice, string(target.Direction),
		target.CreatedAt, string(target.LastState), string(target.Type), target.TrailingPercent,
		target.TrailingValue, target.ActivationPrice, isActiveInt, target.ExtremePrice, target.Note,
		target.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update target: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected during update: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("target not found")
	}

	return nil
}

func (s *SQLiteStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `DELETE FROM targets WHERE id = ?`

	res, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected during delete: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("target not found")
	}

	return nil
}
