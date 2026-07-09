// controller/BotHandler.go
package controller

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CommandResult resultado estructurado de la ejecución de un comando
type CommandResult struct {
	Output          string        // stdout + stderr combinados
	ExitCode        int           // código de salida (0 = éxito)
	Err             error         // error de ejecución (si lo hubo)
	Duration        time.Duration // cuánto tardó
	Timeout         bool          // true si se agotó el tiempo
	TimeoutDuration time.Duration
}

// Success retorna true si el comando terminó con código 0 y sin error
func (r *CommandResult) Success() bool {
	return r.Err == nil && r.ExitCode == 0
}

// CommandExecutor servicio para ejecutar comandos del sistema
type CommandExecutor struct {
	// Timeout máximo por defecto para cada comando
	DefaultTimeout time.Duration
	// Longitud máxima del output (para no saturar Telegram)
	MaxOutputLength int
	// Logger opcional (puede ser nil)
	Log *Log
	// Acceso a shell aliases
	config *Config
}

// NewCommandExecutor crea un executor con valores por defecto
func NewCommandExecutor(config *Config) *CommandExecutor {
	return &CommandExecutor{
		DefaultTimeout:  30 * time.Second,
		MaxOutputLength: 4000,
		Log:             config.Log,
		config:          config,
	}
}

// Execute ejecuta un comando con el timeout por defecto
func (e *CommandExecutor) Execute(command string) *CommandResult {
	ctx, cancel := context.WithTimeout(context.Background(), e.DefaultTimeout)
	defer cancel()
	return e.ExecuteContext(ctx, command)
}

// ExecuteContext ejecuta un comando con un contexto personalizado
func (e *CommandExecutor) ExecuteContext(ctx context.Context, command string) *CommandResult {
	result := &CommandResult{}

	command = strings.TrimSpace(command)
	if command == "" {
		result.Err = fmt.Errorf("comando vacío")
		result.ExitCode = -1
		return result
	}

	// ← NUEVO: Normalizar primera letra a minúscula
	if len(command) > 0 {
		firstChar := strings.ToLower(string(command[0]))
		command = firstChar + command[1:]
	}

	if e.Log != nil {
		e.Log.Comentario("INFO", fmt.Sprintf("Ejecutando: %s", command))
	}

	cmd := e.buildCommand(command)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	start := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(start)

	result.Output = e.truncateOutput(out.String())

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Timeout = true
			result.TimeoutDuration = e.DefaultTimeout
			result.Err = fmt.Errorf("timeout: el comando tardó más de %v", e.DefaultTimeout)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Err = err
		}
	}

	if e.Log != nil {
		if result.Success() {
			e.Log.Comentario("SUCCESS", fmt.Sprintf("OK en %v", result.Duration))
		} else {
			e.Log.Comentario("WARNING", fmt.Sprintf("Falló (exit=%d, err=%v)", result.ExitCode, result.Err))
		}
	}

	return result
}

// buildCommand crea el exec.Cmd correcto según el sistema operativo
func (e *CommandExecutor) buildCommand(command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/C", command)
	}

	// Si no hay aliases, ejecutar directamente con sh
	if len(e.config.ShellAliases) == 0 {
		return exec.Command("sh", "-c", command)
	}

	// Detectar el shell disponible
	shell := detectAvailableShell()
	loader := buildAliasLoader(shell, e.config.ShellAliases, command)

	return exec.Command(shell, "-c", loader)
}

// detectAvailableShell detecta qué shell está disponible en el sistema
func detectAvailableShell() string {
	// Lista de shells en orden de preferencia
	shells := []string{"/bin/bash", "/bin/zsh", "/bin/ash", "/bin/sh"}

	for _, shell := range shells {
		if _, err := os.Stat(shell); err == nil {
			return shell
		}
	}

	// Fallback a sh que debería existir en todos los sistemas POSIX
	return "/bin/sh"
}

// buildAliasLoader construye el script para cargar aliases según el shell
func buildAliasLoader(shell string, aliases map[string]string, command string) string {
	var sb strings.Builder

	// Determinar el tipo de shell por el nombre
	shellName := filepath.Base(shell)

	switch shellName {
	case "bash":
		sb.WriteString("shopt -s expand_aliases\n")
		for name, cmd := range aliases {
			escapedCmd := escapeShellArg(cmd)
			sb.WriteString(fmt.Sprintf("alias %s='%s'\n", name, escapedCmd))
		}

	case "zsh":
		sb.WriteString("setopt aliases\n")
		for name, cmd := range aliases {
			escapedCmd := escapeShellArg(cmd)
			sb.WriteString(fmt.Sprintf("alias %s='%s'\n", name, escapedCmd))
		}

	default: // ash, sh, y otros shells POSIX
		// Usamos funciones como alternativa
		for name, cmd := range aliases {
			escapedCmd := escapeShellArg(cmd)
			// Crear una función que imite el comportamiento del alias
			sb.WriteString(fmt.Sprintf("%s() { %s \"$@\"; }\n", name, escapedCmd))
		}
	}

	sb.WriteString("\n")
	sb.WriteString(command)

	return sb.String()
}

// escapeShellArg escapa comillas simples en argumentos shell
func escapeShellArg(arg string) string {
	// Reemplazar ' con '\'' para escapar en shell
	return strings.ReplaceAll(arg, "'", "'\"'\"'")
}

// truncateOutput limita el tamaño del output para no saturar Telegram
func (e *CommandExecutor) truncateOutput(output string) string {
	if len(output) <= e.MaxOutputLength {
		return output
	}
	return output[:e.MaxOutputLength] + "\n\n... [truncado]"
}

// FormatForTelegram formatea el resultado para enviarlo por Telegram (Markdown)
func (r *CommandResult) FormatForTelegram(command string) string {
	var sb strings.Builder

	sb.WriteString("💻 *Ejecutando:* `")
	sb.WriteString(command)
	sb.WriteString("`\n\n")

	if r.Timeout {
		sb.WriteString(fmt.Sprintf("⏰ *Timeout* (>%v)\n", r.TimeoutDuration))
	}

	if r.Output != "" {
		sb.WriteString("```\n")
		sb.WriteString(r.Output)
		if !strings.HasSuffix(r.Output, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n")
	}

	if !r.Success() && r.Err != nil {
		sb.WriteString(fmt.Sprintf("\n❌ *Error:* %s", r.Err.Error()))
	} else if r.ExitCode != 0 {
		sb.WriteString(fmt.Sprintf("\n⚠️ *Exit code:* %d", r.ExitCode))
	} else {
		sb.WriteString(fmt.Sprintf("\n✅ *OK* en %v", r.Duration))
	}

	return sb.String()
}
