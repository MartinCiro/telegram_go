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

	// 1️⃣ Instanciar configuración (ella sola carga .env y valida)
	config := controller.NewConfig()

	// 2️⃣ Instanciar servicios con inyección de dependencias
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

	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Información completa del bot"},
		{Command: "estado", Description: "Ver IPs y red actual"},
		{Command: "comando", Description: "Ejecutar comando del sistema"},
		{Command: "ayuda", Description: "Lista de comandos"},
	}

	setCmd := tgbotapi.NewSetMyCommands(commands...)
	if _, err := bot.Request(setCmd); err != nil {
		config.Log.Error(fmt.Sprintf("Error registrando comandos: %v", err), "Telegram")
	} else {
		config.Log.Comentario("SUCCESS", "Menú de comandos registrado")
	}

	bot.Debug = false
	config.Log.Comentario("SUCCESS", fmt.Sprintf("Bot autenticado como @%s", bot.Self.UserName))
	fmt.Printf("✅ Bot autenticado como @%s\n", bot.Self.UserName)

	// 4️⃣ Obtener info de red para notificación inicial
	info := networkService.ObtenerInfo()
	fmt.Printf("🌐 IP Pública: %s\n", info.IPPublica)
	fmt.Printf("🏠 IP Local: %s\n", info.IPLocal)
	fmt.Printf("📡 Red: %s\n", info.Red)
	fmt.Println("=" + "==================================================")

	// 5️⃣ Enviar notificación inicial si hay chat ID configurado
	if config.TelegramChat != "" {
		var chatID int64
		if _, err := fmt.Sscanf(config.TelegramChat, "%d", &chatID); err == nil {
			notificacion := fmt.Sprintf(
				"🤖 *Bot Iniciado*\n\n"+
					"✅ Sistema activo y listo\n\n"+
					"%s",
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

	// 6️⃣ Configurar polling de updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	config.Log.Comentario("INFO", "Esperando mensajes...")
	fmt.Println("📱 Esperando mensajes... (Ctrl+C para salir)")

	// 7️⃣ Manejo de shutdown graceful
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 8️⃣ Loop principal de procesamiento
	go func() {
		for update := range updates {
			if update.Message == nil {
				continue
			}

			config.Log.Comentario("INFO", fmt.Sprintf("Mensaje de %s: %s",
				update.Message.From.UserName, update.Message.Text))

			// Delegar al handler
			respuesta := handler.Handle(update.Message.Text)

			// Enviar respuesta
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, respuesta)
			msg.ParseMode = "Markdown"

			if _, err := bot.Send(msg); err != nil {
				config.Log.Error(fmt.Sprintf("Error enviando respuesta: %v", err), "Telegram")
			}
		}
	}()

	// 9️⃣ Esperar señal de terminación
	<-sigChan
	config.Log.Comentario("INFO", "Recibida señal de terminación")
	config.Log.FinProceso("Bot Telegram")
	fmt.Println("\n🛑 Bot detenido")
}
