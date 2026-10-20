package db

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open estabelece e retorna uma conexão *gorm.DB com base nas variáveis de ambiente.
func Open() (*gorm.DB, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("DB_URL")
	}

	var dialector gorm.Dialector

	if databaseURL != "" {
		d, err := dialectorFromURL(databaseURL)
		if err != nil {
			return nil, err
		}
		dialector = d
	} else {
		driver := strings.ToLower(getEnvOrDefault("DB_DRIVER", getEnvOrDefault("DB_CONNECTION", "postgres")))
		switch driver {
		case "postgres", "postgresql", "pgx":
			host := getEnvOrDefault("DB_HOST", "localhost")
			port := getEnvOrDefault("DB_PORT", "5432")
			user := getEnvOrDefault("DB_USER", getEnvOrDefault("DB_USERNAME", "postgres"))
			password := getEnvOrDefault("DB_PASSWORD", getEnvOrDefault("DB_PASS", ""))
			dbname := getEnvOrDefault("DB_NAME", getEnvOrDefault("DB_DATABASE", "postgres"))
			sslmode := getEnvOrDefault("DB_SSLMODE", "disable")
			timezone := getEnvOrDefault("DB_TIMEZONE", "America/Sao_Paulo")

			dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
				host, user, password, dbname, port, sslmode, timezone)
			dialector = postgres.Open(dsn)

		case "mysql":
			host := getEnvOrDefault("DB_HOST", "127.0.0.1")
			port := getEnvOrDefault("DB_PORT", "3306")
			user := getEnvOrDefault("DB_USER", getEnvOrDefault("DB_USERNAME", "root"))
			password := getEnvOrDefault("DB_PASSWORD", getEnvOrDefault("DB_PASS", ""))
			dbname := getEnvOrDefault("DB_NAME", getEnvOrDefault("DB_DATABASE", ""))

			dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				user, password, host, port, dbname)
			dialector = mysql.Open(dsn)

		case "sqlite", "sqlite3":
			dbname := getEnvOrDefault("DB_NAME", getEnvOrDefault("DB_DATABASE", "gmt.db"))
			dialector = sqlite.Open(dbname)

		default:
			return nil, fmt.Errorf("driver de banco de dados não suportado: %s", driver)
		}
	}

	logLevel := logger.Warn
	if strings.ToLower(os.Getenv("GMT_DEBUG")) == "true" {
		logLevel = logger.Info
	}

	return gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
}

func dialectorFromURL(rawURL string) (gorm.Dialector, error) {
	if strings.HasPrefix(rawURL, "sqlite://") {
		return sqlite.Open(strings.TrimPrefix(rawURL, "sqlite://")), nil
	}
	if strings.HasSuffix(rawURL, ".db") || strings.HasSuffix(rawURL, ".sqlite") {
		return sqlite.Open(rawURL), nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("URL de banco de dados inválida: %w", err)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "postgres", "postgresql":
		return postgres.Open(rawURL), nil
	case "mysql":
		// Formata para mysql DSN
		user := parsed.User.Username()
		password, _ := parsed.User.Password()
		host := parsed.Host
		dbname := strings.TrimPrefix(parsed.Path, "/")
		query := parsed.RawQuery
		if query != "" {
			query = "?" + query
		} else {
			query = "?charset=utf8mb4&parseTime=True&loc=Local"
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s%s", user, password, host, dbname, query)
		return mysql.Open(dsn), nil
	default:
		return nil, fmt.Errorf("esquema de URL de banco não suportado: %s", parsed.Scheme)
	}
}

// InitDB conecta ao banco ou encerra a aplicação com mensagem explicativa se houver erro.
func InitDB() *gorm.DB {
	db, err := Open()
	if err != nil {
		log.Fatalf("❌ Erro ao conectar ao banco de dados: %v\nVerifique se o arquivo .env ou as variáveis de ambiente (DB_HOST, DB_USER, etc) estão configuradas corretamente.", err)
	}
	return db
}

func getEnvOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
