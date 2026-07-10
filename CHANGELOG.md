## [v1.0.3] - 2026-07-11

### Añadido
- Descarte automático de mensajes pendientes al iniciar el bot
- Configuración `DropPendingUpdates` para evitar ejecución de comandos antiguos
- Script de release con compilación automática de binarios si no existen
- Detección de cambios de versión para recompilación automática

### Mejorado
- Experiencia de usuario: el bot ahora solo responde a mensajes enviados después de su inicio
- Limpieza automática de la cola de updates pendientes de Telegram
- Script de release: compilación cruzada automática (linux/amd64, linux/arm64, windows/amd64)
- Script de release: tracking de versión para evitar recompilaciones innecesarias

### Corregido
- **IMPORTANTE**: Comandos enviados con el bot apagado ya no se ejecutan al reiniciar
- **IMPORTANTE**: Cola de mensajes acumulados se descarta automáticamente al iniciar

### Cambiado
- `main.go`: Uso de `DeleteWebhookConfig` con `DropPendingUpdates: true` antes de iniciar polling
- `main.go`: Eliminada lógica manual de offset con `GetUpdates` (reemplazada por descarte nativo de Telegram)
- `release.sh`: Compilación automática de binarios si no existen o si la versión cambió
- `release.sh`: Limpieza de binarios anteriores cuando se detecta cambio de versión

### Técnico
- Implementación de `tgbotapi.DeleteWebhookConfig` para limpieza de cola
- Polling ahora inicia con cola vacía garantizada
- Script de release usa `build/.last_version` para tracking de compilaciones
- Compilación cruzada con `GOOS` y `GOARCH` para múltiples plataformas

---