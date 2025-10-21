package postgres

import (
	"database/sql"
	"strconv"
	"sync"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config/db"
)

type PostgresStorage struct {
	db      *sql.DB
	baseURL string
	mu      sync.RWMutex
	counter int64
}

const upsertURLSQL = `
	INSERT INTO short_urls (id, short_url, original_url) 
	VALUES ($1, $2, $3)
	ON CONFLICT (id) 
	DO UPDATE SET 
		original_url = EXCLUDED.original_url,
		is_deleted = FALSE
`

func NewPostgresStorage(DB *sql.DB, baseURL string) (*PostgresStorage, error) {
	if err := db.RunMigrations(DB); err != nil {
		return nil, err
	}

	// Восстанавливаем счетчик из БД
	var maxCounter sql.NullInt64
	DB.QueryRow(`SELECT MAX(CAST(id AS BIGINT)) FROM short_urls`).Scan(&maxCounter)

	counter := int64(0)
	if maxCounter.Valid {
		counter = maxCounter.Int64
	}

	return &PostgresStorage{
		db:      DB,
		baseURL: baseURL,
		counter: counter,
	}, nil
}

func (p *PostgresStorage) Save(originalURL string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Сначала проверяем, нет ли уже такого URL в БД
	var existingID string
	err := p.db.QueryRow(
		`SELECT id FROM short_urls WHERE original_url = $1 AND is_deleted = false`,
		originalURL,
	).Scan(&existingID)

	// Если URL уже существует - возвращаем существующий ID
	if err == nil {
		return existingID
	}

	// Если это новый URL - создаем новую запись
	p.counter++
	id := strconv.FormatInt(p.counter, 10)
	shortURL := p.baseURL + "/" + id

	_, err = p.db.Exec(upsertURLSQL, id, shortURL, originalURL)
	if err != nil {
		return ""
	}

	return id
}
func (p *PostgresStorage) Get(id string) (string, bool) {
	var originalURL string
	err := p.db.QueryRow(
		`SELECT "original_url" FROM "short_urls" WHERE "id" = $1 AND "is_deleted" = false`,
		id,
	).Scan(&originalURL)

	if err != nil {
		return "", false
	}
	return originalURL, true
}
