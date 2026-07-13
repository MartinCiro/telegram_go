// controller/BotHandler.go
package controller

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// BotHandler orquestador que rutea mensajes a los servicios correctos
type BotHandler struct {
	network         *NetworkService
	executor        *CommandExecutor
	updater         *UpdateService
	tunnel          *TunnelService
	log             *Log
	pendingCommands map[int64]time.Time
	mu              sync.Mutex
}

// NewBotHandler crea un handler con dependencias inyectadas
func NewBotHandler(network *NetworkService, executor *CommandExecutor, updater *UpdateService, tunnel *TunnelService, log *Log) *BotHandler {
	return &BotHandler{
		network:         network,
		executor:        executor,
		updater:         updater,
		tunnel:          tunnel,
		log:             log,
		pendingCommands: make(map[int64]time.Time),
	}
}

// Handle procesa un mensaje de texto y retorna la respuesta
func (h *BotHandler) Handle(chatID int64, text string) *Response {
	text = strings.TrimSpace(text)
	if text == "" {
		return NewResponse("❓ Mensaje vacío")
	}

	// PRIORIDAD 1: Si el chat está esperando comando
	h.mu.Lock()
	pendingTime, waiting := h.pendingCommands[chatID]

	// Limpiar si expiró (5 minutos)
	if waiting && time.Since(pendingTime) > 5*time.Minute {
		delete(h.pendingCommands, chatID)
		waiting = false
		if h.log != nil {
			h.log.Comentario("WARNING", fmt.Sprintf("⏰ Timeout de comando pendiente para chat %d", chatID))
		}
	}

	if waiting {
		delete(h.pendingCommands, chatID)
	} else if strings.ToLower(text) == "/comando" {
		h.pendingCommands[chatID] = time.Now() // ← Guardar timestamp
	}
	h.mu.Unlock()

	if waiting {
		return h.handleComando(text)
	}

	// PRIORIDAD 2: Ruteo normal
	lower := strings.ToLower(text)

	switch {
	case lower == "/start" || lower == "/inicio" || lower == "🏠":
		return h.handleStart()

	case lower == "/estado" || lower == "ℹ️":
		return h.handleEstado()

	case lower == "/ayuda" || lower == "/help" || lower == "❓":
		return h.handleAyuda()

	case lower == "/up" || lower == "🔄":
		return h.handleUpdate()

	case lower == "/ver_url" || lower == "🔗":
		return h.handleVerUrl()

	case lower == "/comando" || lower == "💻":
		h.mu.Lock()
		h.pendingCommands[chatID] = time.Now()
		h.mu.Unlock()

		resp := NewResponse("💻 Envíame el comando que quieres ejecutar:")
		resp.ForceReply = true
		return resp

	case strings.HasPrefix(lower, "/comando "):
		cmd := strings.TrimPrefix(text, "/comando ")
		cmd = strings.TrimSpace(cmd)
		return h.handleComando(cmd)

	default:
		return NewResponse("❓ Comando no reconocido. Usa /ayuda para ver los comandos disponibles.")
	}
}

// handleStart retorna información completa del sistema
func (h *BotHandler) handleStart() *Response {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /start ejecutado")
	}

	info := h.network.ObtenerInfo()
	text := fmt.Sprintf(
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

	return NewResponse(text).WithButtons(
		Button{Text: "📊 Ver Estado", Data: "/estado", Type: ButtonInline},
		Button{Text: "❓ Ayuda", Data: "/ayuda", Type: ButtonInline},
	)
}

// handleEstado retorna solo información de red
func (h *BotHandler) handleEstado() *Response {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /estado ejecutado")
	}

	info := h.network.ObtenerInfo()
	text := fmt.Sprintf(
		"✅ *Bot en línea*\n\n%s",
		info.FormatearParaTelegram(),
	)

	return NewResponse(text)
}

// handleAyuda retorna la lista de comandos disponibles
func (h *BotHandler) handleAyuda() *Response {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /ayuda ejecutado")
	}

	text := "📋 *Comandos disponibles:*\n\n" +
		"• /start o /inicio - Muestra información completa del bot\n" +
		"• /estado - Ver estado actual e IPs\n" +
		"• /comando <cmd> - Ejecutar comando del sistema\n" +
		"• /ayuda - Esta ayuda"

	return NewResponse(text).WithButtons(
		Button{Text: "🏠 Inicio", Data: "/start", Type: ButtonInline},
		Button{Text: "📊 Estado", Data: "/estado", Type: ButtonInline},
	)
}

// handleComando ejecuta un comando del sistema y retorna el resultado
func (h *BotHandler) handleComando(command string) *Response {
	if command == "" {
		return NewResponse("❌ Debes especificar un comando. Ejemplo: `/comando ls -la`")
	}

	if h.log != nil {
		h.log.Comentario("INFO", fmt.Sprintf("Ejecutando comando: %s", command))
	}

	result := h.executor.Execute(command)
	text := result.FormatForTelegram(command)

	return NewResponse(text)
}

// handleUpdate descarga la última versión del bot
func (h *BotHandler) handleUpdate() *Response {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /up ejecutado - Iniciando actualización")
	}

	result := h.updater.CheckAndUpdate()
	text := result.FormatForTelegram()

	return NewResponse(text)
}

// handleVerUrl muestra la URL del túnel
func (h *BotHandler) handleVerUrl() *Response {
	if h.log != nil {
		h.log.Comentario("INFO", "Comando /ver_url ejecutado")
	}

	// El servicio se encarga de verificar, iniciar si es necesario y retornar la URL
	url, err := h.tunnel.EnsureTunnelRunning()
	if err != nil {
		return NewResponse(fmt.Sprintf("❌ *Error:* %v\n\n💡 Asegúrate de que `cloudflared` esté instalado en el sistema.", err))
	}

	cleanURL := strings.TrimPrefix(url, "https://")

	return NewResponse(cleanURL)
}
