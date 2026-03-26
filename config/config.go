package config

import (
	"flag"
	"fmt"
)

type Config struct {
	Port    int
	TLSCert string
	TLSKey  string
	DBPath  string
}

func (c *Config) TLSEnabled() bool {
	return c.TLSCert != "" && c.TLSKey != ""
}

func (c *Config) ListenAddr() string {
	return fmt.Sprintf(":%d", c.Port)
}

func Parse() *Config {
	cfg := &Config{}
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&cfg.TLSCert, "tls-cert", "", "Path to TLS certificate file (enables TLS when set with -tls-key)")
	flag.StringVar(&cfg.TLSKey, "tls-key", "", "Path to TLS private key file (enables TLS when set with -tls-cert)")
	flag.StringVar(&cfg.DBPath, "db", "sickgnal.db", "Path to SQLite database file")
	flag.Parse()
	return cfg
}
