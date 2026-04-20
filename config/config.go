package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Port    int
	TLSCert string
	TLSKey  string

	// Database
	DBDriver    string // "sqlite" or "postgres"
	DBPath      string // SQLite file path
	DatabaseURL string // Full PostgreSQL DSN (for dev convenience)
	DBHost      string
	DBPort      int
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
}

func (c *Config) TLSEnabled() bool {
	return c.TLSCert != "" && c.TLSKey != ""
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}

// DSN resolves the PostgreSQL connection string.
// Priority: separated env vars > DATABASE_URL > error.
func (c *Config) DSN() (string, error) {
	// If individual fields are provided (at minimum DB_HOST or DB_PASSWORD set via env),
	// build the DSN from separated variables.
	if c.DBHost != "localhost" || c.DBPassword != "" || c.DBUser != "sickgnal" || c.DBName != "sickgnal" {
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode), nil
	}

	// Fallback to DATABASE_URL (full DSN, e.g. postgres://user:pass@host:5432/db?sslmode=disable)
	if c.DatabaseURL != "" {
		return c.DatabaseURL, nil
	}

	return "", fmt.Errorf("postgres driver selected but no connection info provided: set DB_HOST/DB_PASSWORD env vars or DATABASE_URL")
}

func Parse() *Config {
	cfg := &Config{}

	// Define CLI flags
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&cfg.TLSCert, "tls-cert", "", "Path to TLS certificate file")
	flag.StringVar(&cfg.TLSKey, "tls-key", "", "Path to TLS private key file")
	flag.StringVar(&cfg.DBDriver, "db-driver", "sqlite", "Database driver: sqlite or postgres")
	flag.StringVar(&cfg.DBPath, "db", "sickgnal.db", "Path to SQLite database file")
	flag.StringVar(&cfg.DatabaseURL, "database-url", "", "Full PostgreSQL DSN (e.g. postgres://user:pass@host:5432/db?sslmode=disable)")
	flag.StringVar(&cfg.DBHost, "db-host", "localhost", "PostgreSQL host")
	flag.IntVar(&cfg.DBPort, "db-port", 5432, "PostgreSQL port")
	flag.StringVar(&cfg.DBUser, "db-user", "sickgnal", "PostgreSQL user")
	flag.StringVar(&cfg.DBPassword, "db-password", "", "PostgreSQL password")
	flag.StringVar(&cfg.DBName, "db-name", "sickgnal", "PostgreSQL database name")
	flag.StringVar(&cfg.DBSSLMode, "db-sslmode", "disable", "PostgreSQL SSL mode")
	flag.Parse()

	// Environment variables override CLI flags
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Port = p
		}
	}
	if v := os.Getenv("TLS_CERT"); v != "" {
		cfg.TLSCert = v
	}
	if v := os.Getenv("TLS_KEY"); v != "" {
		cfg.TLSKey = v
	}
	if v := os.Getenv("DB_DRIVER"); v != "" {
		cfg.DBDriver = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseURL = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.DBHost = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.DBPort = p
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.DBUser = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.DBPassword = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.DBSSLMode = v
	}

	return cfg
}
