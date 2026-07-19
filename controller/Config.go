// controller/Config.go
package controller

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config configuración del bot
type Config struct {
	TelegramToken string
	TelegramChat  string
	AllowedUsers  []int64
	ShellAliases  map[string]string
	Log           *Log
	WolTargetIP   string
	WolTargetUser string
	WolSSHPort    int
	WolTargetMAC  string
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

	allowedUsers := parseAllowedUsers(os.Getenv("ALLOWED_USERS"))
	shellAliases := parseShellAliases(os.Getenv("SHELL_ALIASES"))

	// Añadir TELEGRAM_CHAT automáticamente a la lista blanca
	if chatID != "" {
		if id, err := strconv.ParseInt(chatID, 10, 64); err == nil {
			if !containsInt64(allowedUsers, id) {
				allowedUsers = append(allowedUsers, id)
				fmt.Printf("✅ TELEGRAM_CHAT (%d) añadido automáticamente a ALLOWED_USERS\n", id)
			}
		}
	}

	wolIP := os.Getenv("WOL_TARGET_IP")
	if wolIP == "" {
		wolIP = "192.168.0.61"
	}

	wolUser := os.Getenv("WOL_TARGET_USER")
	if wolUser == "" {
		wolUser = "user"
	}

	wolPort := 22
	if portStr := os.Getenv("WOL_SSH_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			wolPort = p
		}
	}

	wolMAC := os.Getenv("WOL_TARGET_MAC")

	return &Config{
		TelegramToken: token,
		TelegramChat:  chatID,
		AllowedUsers:  allowedUsers,
		ShellAliases:  shellAliases,
		Log:           NewLog(),
		WolTargetIP:   wolIP,
		WolTargetUser: wolUser,
		WolSSHPort:    wolPort,
		WolTargetMAC:  strings.TrimSpace(wolMAC),
	}
}

func parseShellAliases(raw string) map[string]string {
	aliases := make(map[string]string)

	if strings.TrimSpace(raw) == "" {
		return aliases
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			fmt.Printf("⚠️  SHELL_ALIASES: formato inválido '%s' (esperado: name=command)\n", part)
			continue
		}

		name := strings.TrimSpace(kv[0])
		command := strings.TrimSpace(kv[1])

		if name == "" || command == "" {
			fmt.Printf("⚠️  SHELL_ALIASES: nombre o comando vacío en '%s'\n", part)
			continue
		}

		aliases[name] = command
	}

	if len(aliases) > 0 {
		fmt.Printf("✅ Cargados %d shell aliases\n", len(aliases))
	}

	return aliases
}

// containsInt64 verifica si un slice contiene un valor
func containsInt64(slice []int64, value int64) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

func parseAllowedUsers(raw string) []int64 {
	if strings.TrimSpace(raw) == "" {
		return nil // nil = sin restricción (permite a todos)
	}

	var users []int64
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			fmt.Printf("⚠️  ALLOWED_USERS: ignorando valor inválido '%s'\n", p)
			continue
		}
		users = append(users, id)
	}
	return users
}

// IsUserAllowed verifica si un usuario está en la lista blanca
// Si la lista está vacía, permite a todos (backward compatible)
func (c *Config) IsUserAllowed(userID int64) bool {
	if len(c.AllowedUsers) == 0 {
		return true // Sin restricción
	}
	for _, allowed := range c.AllowedUsers {
		if allowed == userID {
			return true
		}
	}
	return false
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
