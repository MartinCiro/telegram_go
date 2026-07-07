package controller

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config configuración del bot
type Config struct {
	TelegramToken string
	TelegramChat  string
	Log           *Log
}

// NewConfig crea una nueva instancia de Config
func NewConfig() *Config {
	// Cargar .env si existe
	envPath := ".env"
	if _, err := os.Stat(envPath); err == nil {
		if err := godotenv.Load(); err != nil {
			fmt.Printf("⚠️  Advertencia: No se pudo cargar .env: %v\n", err)
		}
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("TELEGRAM_CHAT")

	if token == "" {
		fmt.Println("❌ ERROR: TELEGRAM_TOKEN no está configurado en .env")
		os.Exit(1)
	}

	return &Config{
		TelegramToken: token,
		TelegramChat:  chatID,
		Log:           NewLog(),
	}
}

// GetProjectPath retorna la ruta absoluta del proyecto
func (c *Config) GetProjectPath() string {
	path, err := os.Getwd()
	if err != nil {
		return "."
	}
	return path
}

// GetLogPath retorna la ruta del directorio de logs
func (c *Config) GetLogPath() string {
	return filepath.Join(c.GetProjectPath(), "logs")
}
