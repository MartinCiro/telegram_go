package controller

import (
	"fmt"
	"runtime"
	"strings"
)

// BotHandler orquestador que rutea mensajes a los servicios correctos
type BotHandler struct {
	network  *NetworkService
	executor *CommandExecutor
	log      *Log
}

// NewBotHandler crea un handler con dependencias inyectadas
func NewBotHandler(network *NetworkService, executor *CommandExecutor, log *Log) *BotHandler {
	return &BotHandler{
		network:  network,
		executor: executor,
		log:      log,
	}
}

// Handle procesa un mensaje de texto y retorna la respuesta
func (h *BotHandler) Handle(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "❓ Mensaje vacío"
	}

	// Convertir a minúsculas para comparación
	lower := strings.ToLower(text)

	// Ruteo de comandos
	switch {
	case lower == "/start" || lower == "/inicio":
		return h.handleStart()

	case lower == "/estado":
		return h.handleEstado()

	case lower == "/ayuda" || lower == "/help":
		return h.handleAyuda()

	case strings.HasPrefix(lower, "/comando "):
		cmd := strings.TrimPrefix(text, "/comando ")
		cmd = strings.TrimPrefix(cmd, "/comando")
		cmd = strings.TrimSpace(cmd)
		return h.handleComando(cmd)

	case strings.HasPrefix(lower, "/comando"):
		return "❌ Debes especificar un comando. Ejemplo: `/comando ls -la`"

	default:
		return "❓ Comando no reconocido. Usa /ayuda para ver los comandos disponibles."
	}
}

// handleStart retorna información completa del sistema
func (h *BotHandler) handleStart() string {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /start ejecutado")
	}

	info := h.network.ObtenerInfo()

	return fmt.Sprintf(
		"🤖 *Bot Activo*\n\n"+
			"📍 *Información de Red:*\n%s\n\n"+
			"🖥️ *Sistema:*\n"+
			"• OS: `%s`\n"+
			"• Arquitectura: `%s`\n\n"+
			"💡 Usa /comando <cmd> para ejecutar comandos del sistema",
		info.FormatearParaTelegram(),
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// handleEstado retorna solo información de red
func (h *BotHandler) handleEstado() string {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /estado ejecutado")
	}

	info := h.network.ObtenerInfo()
	return fmt.Sprintf(
		"✅ *Bot en línea*\n\n%s",
		info.FormatearParaTelegram(),
	)
}

// handleAyuda retorna la lista de comandos disponibles
func (h *BotHandler) handleAyuda() string {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /ayuda ejecutado")
	}

	return "📋 *Comandos disponibles:*\n\n" +
		"• /start o /inicio - Muestra información completa del bot\n" +
		"• /estado - Ver estado actual e IPs\n" +
		"• /comando <cmd> - Ejecutar comando del sistema\n" +
		"• /ayuda - Esta ayuda"
}

// handleComando ejecuta un comando del sistema y retorna el resultado
func (h *BotHandler) handleComando(command string) string {
	if command == "" {
		return "❌ Debes especificar un comando. Ejemplo: `/comando ls -la`"
	}

	if h.log != nil {
		h.log.Comentario("INFO", fmt.Sprintf("Ejecutando comando: %s", command))
	}

	result := h.executor.Execute(command)
	return result.FormatForTelegram(command)
}
