package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go-tel/controller"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	fmt.Println("=" + "==================================================")
	fmt.Println("🤖 Bot de Telegram - Sistema Básico")
	fmt.Println("=" + "==================================================")

	// 1️⃣ Instanciar configuración
	config := controller.NewConfig()

	// 2️⃣ Instanciar servicios
	networkService := controller.NewNetworkService()
	executor := controller.NewCommandExecutor(config.Log)
	handler := controller.NewBotHandler(networkService, executor, config.Log)

	config.Log.InicioProceso("Bot Telegram")
	config.Log.Comentario("SUCCESS", "Servicios inicializados")

	// 3️⃣ Crear bot de Telegram
	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		config.Log.Error(fmt.Sprintf("Error autenticando: %v", err), "Telegram")
		log.Fatalf("❌ Error: %v", err)
	}

	bot.Debug = false
	config.Log.Comentario("SUCCESS", fmt.Sprintf("Bot autenticado como @%s", bot.Self.UserName))
	fmt.Printf("✅ Bot autenticado como @%s\n", bot.Self.UserName)

	// 4️⃣ Registrar menú de comandos (Opción 1)
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Información completa del bot"},
		{Command: "estado", Description: "Ver IPs y red actual"},
		{Command: "comando", Description: "Ejecutar comando del sistema"},
		{Command: "ayuda", Description: "ℹ️ Lista de comandos"},

		{Command: "icono_home", Description: "🏠 Inicio"},
		{Command: "icono_status", Description: "ℹ️ Lista de comandos"},
		{Command: "icono_bash", Description: "💻 Terminal"},
		{Command: "icono_help", Description: "❓ Ayuda"},
	}
	setCmd := tgbotapi.NewSetMyCommands(commands...)
	if _, err := bot.Request(setCmd); err != nil {
		config.Log.Error(fmt.Sprintf("Error registrando comandos: %v", err), "Telegram")
	} else {
		config.Log.Comentario("SUCCESS", "Menú de comandos registrado")
	}

	// 5️⃣ Obtener info de red
	info := networkService.ObtenerInfo()
	fmt.Printf("🌐 IP Pública: %s\n", info.IPPublica)
	fmt.Printf("🏠 IP Local: %s\n", info.IPLocal)
	fmt.Printf("📡 Red: %s\n", info.Red)
	fmt.Println("=" + "==================================================")

	// 6️⃣ Notificación inicial
	if config.TelegramChat != "" {
		var chatID int64
		if _, err := fmt.Sscanf(config.TelegramChat, "%d", &chatID); err == nil {
			notificacion := fmt.Sprintf(
				"🤖 *Bot Iniciado*\n\n✅ Sistema activo y listo\n\n%s",
				info.FormatearParaTelegram(),
			)
			msg := tgbotapi.NewMessage(chatID, notificacion)
			msg.ParseMode = "Markdown"
			if _, err := bot.Send(msg); err != nil {
				config.Log.Error(fmt.Sprintf("Error enviando notificación inicial: %v", err), "Telegram")
			} else {
				config.Log.Comentario("SUCCESS", "Notificación inicial enviada")
			}
		}
	}

	// 7️⃣ Configurar polling
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	config.Log.Comentario("INFO", "Esperando mensajes...")
	fmt.Println("📱 Esperando mensajes... (Ctrl+C para salir)")

	// 8️⃣ Shutdown graceful
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 9️⃣ Loop principal
	go func() {
		for update := range updates {
			// ──────────────────────────────────────────────
			// CASO 1: Callback de botón inline
			// ──────────────────────────────────────────────
			if update.CallbackQuery != nil {
				cb := update.CallbackQuery

				config.Log.Comentario("DEBUG", fmt.Sprintf("Callback data: '%s'", cb.Data))

				// Responder al callback
				callbackAnswer := tgbotapi.NewCallback(cb.ID, "")
				if _, err := bot.Request(callbackAnswer); err != nil {
					fmt.Printf("❌ Error respondiendo callback: %v\n", err)
				}

				// Procesar el comando
				response := handler.Handle(cb.Message.Chat.ID, cb.Data)

				// Editar el mensaje
				edit := tgbotapi.NewEditMessageText(
					cb.Message.Chat.ID,
					cb.Message.MessageID,
					response.Text,
				)
				edit.ParseMode = "Markdown"

				// Reconstruir inline keyboard
				if response.HasInlineButtons() {
					var rows [][]tgbotapi.InlineKeyboardButton
					var currentRow []tgbotapi.InlineKeyboardButton

					for _, btn := range response.Buttons {
						if btn.Type == controller.ButtonInline {
							currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.Data))
							if len(currentRow) == 2 {
								rows = append(rows, currentRow)
								currentRow = nil
							}
						}
					}
					if len(currentRow) > 0 {
						rows = append(rows, currentRow)
					}

					kb := tgbotapi.NewInlineKeyboardMarkup(rows...)
					edit.ReplyMarkup = &kb
				}

				// Enviar el edit
				if _, err := bot.Send(edit); err != nil {
					fmt.Printf("❌ Error editando mensaje: %v\n", err)
					config.Log.Error(fmt.Sprintf("Error editando mensaje: %v", err), "Telegram")
				} else {
					// Forzar que se quite el teclado inline
					emptyKb := tgbotapi.NewInlineKeyboardMarkup()
					edit.ReplyMarkup = &emptyKb
				}
				continue
			}

			// ──────────────────────────────────────────────
			// CASO 2: Mensaje normal (texto, reply buttons, etc.)
			// ──────────────────────────────────────────────
			if update.Message == nil {
				continue
			}

			config.Log.Comentario("INFO", fmt.Sprintf("Mensaje de %s: %s",
				update.Message.From.UserName, update.Message.Text))

			response := handler.Handle(update.Message.Chat.ID, update.Message.Text)

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, response.Text)
			msg.ParseMode = "Markdown"

			// Inline keyboard
			if response.HasInlineButtons() {
				var rows [][]tgbotapi.InlineKeyboardButton
				var currentRow []tgbotapi.InlineKeyboardButton

				for _, btn := range response.Buttons {
					if btn.Type == controller.ButtonInline {
						currentRow = append(currentRow, tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.Data))
						if len(currentRow) == 2 {
							rows = append(rows, currentRow)
							currentRow = nil
						}
					}
				}
				if len(currentRow) > 0 {
					rows = append(rows, currentRow)
				}

				msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(rows...)
			}

			persistentButtons := [][]tgbotapi.KeyboardButton{
				{
					tgbotapi.NewKeyboardButton("🏠"),
					tgbotapi.NewKeyboardButton("❓"),
					tgbotapi.NewKeyboardButton("💻"),
					tgbotapi.NewKeyboardButton("ℹ️"),
				},
			}
			persistentKeyboard := tgbotapi.NewReplyKeyboard(persistentButtons...)
			persistentKeyboard.ResizeKeyboard = true

			// Reply keyboard
			if response.HasReplyButtons() {
				var extraRows [][]tgbotapi.KeyboardButton
				var currentRow []tgbotapi.KeyboardButton

				for _, btn := range response.Buttons {
					if btn.Type == controller.ButtonReply {
						currentRow = append(currentRow, tgbotapi.NewKeyboardButton(btn.Text))
						if len(currentRow) == 2 {
							extraRows = append(extraRows, currentRow)
							currentRow = nil
						}
					}
				}
				if len(currentRow) > 0 {
					extraRows = append(extraRows, currentRow)
				}

				// Combinar: botones persistentes + botones de la respuesta
				allRows := append(persistentButtons, extraRows...)
				persistentKeyboard = tgbotapi.NewReplyKeyboard(allRows...)
				persistentKeyboard.ResizeKeyboard = true
			}

			if response.ForceReply {
				msg.ReplyMarkup = tgbotapi.ForceReply{
					ForceReply: true,
					Selective:  true,
				}
			} else {
				msg.ReplyMarkup = persistentKeyboard
			}

			if _, err := bot.Send(msg); err != nil {
				config.Log.Error(fmt.Sprintf("Error enviando respuesta: %v", err), "Telegram")
			}
		}
	}()

	// 🔟 Esperar señal
	<-sigChan
	config.Log.Comentario("INFO", "Recibida señal de terminación")
	config.Log.FinProceso("Bot Telegram")
	fmt.Println("\n🛑 Bot detenido")
}
