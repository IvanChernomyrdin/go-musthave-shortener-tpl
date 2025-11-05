package postgres

import (
	"database/sql"
	"strconv"
	"sync"

	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/config/db"
	"github.com/IvanChernomyrdin/go-musthave-shortener-tpl/internal/storage"
	"github.com/lib/pq"
)

type PostgresStorage struct {
	db      *sql.DB
	baseURL string
	mu      sync.RWMutex
	counter int64
}

const insertURLSQL = `
	INSERT INTO short_urls (id, short_url, original_url, user_id) 
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (original_url) 
	DO NOTHING
	RETURNING id
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

func (p *PostgresStorage) Save(originalURL, userID string) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.counter++
	id := strconv.FormatInt(p.counter, 10)
	shortURL := p.baseURL + "/" + id

	var insertedID string
	err := p.db.QueryRow(insertURLSQL, id, shortURL, originalURL, userID).Scan(&insertedID)

	// Если ошибка "no rows" - значит конфликт (ON CONFLICT DO NOTHING)
	if err == sql.ErrNoRows {
		// Находим существующий URL
		var existingID string
		p.db.QueryRow(
			`SELECT id FROM short_urls WHERE original_url = $1`,
			originalURL,
		).Scan(&existingID)
		return existingID, true // конфликт
	}

	if err != nil {
		return "", false
	}

	return insertedID, false // успешное создание
}

func (p *PostgresStorage) Get(id string) (string, bool) {
	var originalURL string
	var isDeleted bool
	err := p.db.QueryRow(
		`SELECT original_url, is_deleted FROM short_urls WHERE id = $1 AND is_deleted = false`,
		id,
	).Scan(&originalURL, &isDeleted)

	if err != nil {
		return "", false
	}
	if isDeleted {
		return "", true
	}
	return originalURL, true
}

func (p *PostgresStorage) GetURLByUser(userID string) ([]storage.OriginalAndShortURLs, error) {
	rows, err := p.db.Query(`SELECT short_url, original_url FROM short_urls WHERE user_id = $1 AND is_deleted = false`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []storage.OriginalAndShortURLs

	for rows.Next() {
		var shortURL, originalURL string
		if err := rows.Scan(&shortURL, &originalURL); err != nil {
			return nil, err
		}
		urls = append(urls, storage.OriginalAndShortURLs{
			OriginalURL: originalURL,
			ShortURL:    shortURL,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}

func (p *PostgresStorage) DeleteURLs(userID string, urlIDs []string) error {
	if len(urlIDs) == 0 {
		return nil
	}

	query := `UPDATE short_urls 
			  SET is_deleted = TRUE
			  WHERE user_id = $1
			  	AND id = ANY($2) 
				AND is_deleted = FALSE`
	_, err := p.db.Exec(query, userID, pq.Array(urlIDs))
	return err
}
