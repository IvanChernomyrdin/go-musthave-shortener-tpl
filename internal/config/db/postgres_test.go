package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	t.Run("init with valid DSN", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()

		testDSN := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
		if os.Getenv("TEST_TEST_DSN") != "" {
			testDSN = os.Getenv("TEST_TEST_DSN")
		}

		err := Init(testDSN)

		assert.NoError(t, err)
		assert.NotNil(t, DB)
	})
	t.Run("init with empty dns use default", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()

		err := Init("")

		t.Skip("flag and env dsn postgres epmty, init return: %w", err)
	})
	t.Run("double init", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()

		testDSN := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

		err1 := Init(testDSN)
		if err1 != nil {
			t.Skip("Databse not available")
		}
		err2 := Init(testDSN)
		assert.NoError(t, err2)
	})
}

func TestGetConnect(t *testing.T) {
	t.Run("with flag value", func(t *testing.T) {
		result := getConnect("flag_connection")
		assert.Equal(t, "flag_connection", result)
	})

	t.Run("with quoted flag", func(t *testing.T) {
		result := getConnect(`"quoted_connection"`)
		assert.Equal(t, "quoted_connection", result)
	})

	t.Run("default when empty", func(t *testing.T) {
		result := getConnect("")
		assert.Contains(t, result, "postgres://")
	})
}

func TestPing(t *testing.T) {
	t.Run("ping without init", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()
		DB = nil

		err := Ping()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "подключение к базе данных postgres отсутствует")
	})
	t.Run("ping after close", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()

		testDSN := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
		err := Init(testDSN)
		if err != nil {
			t.Skip("Database not available")
		}
		DB.Close()
		DB = nil

		err = Ping()
		assert.Error(t, err)
	})
	t.Run("ping with valid connection", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()

		testDSN := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
		err := Init(testDSN)
		assert.NoError(t, err)

		err = Ping()
		assert.NoError(t, err)
	})
}

func TestIntegration(t *testing.T) {
	t.Run("full lifeCycle", func(t *testing.T) {
		originalDB := DB
		defer func() { DB = originalDB }()

		//делаем init DB не должен быть пустым
		testDSN := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
		err := Init(testDSN)
		if err != nil {
			t.Skip("Database not available")
		}
		require.NotNil(t, DB)

		//пингуем
		err = Ping()
		assert.NoError(t, err)

		//выполняем простой запрос
		var result int
		err = DB.QueryRow(`SELECT 1`).Scan(&result)
		assert.NoError(t, err)
		assert.Equal(t, 1, result)
	})
}
