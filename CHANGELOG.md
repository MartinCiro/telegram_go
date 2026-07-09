# Changelog

Todos los cambios notables en este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/),
y este proyecto sigue [Semantic Versioning](https://semver.org/lang/es/).

## [v1.0.1] - 2026-07-09

### Añadido
- Sistema de auto-actualización desde GitHub Releases
- Nuevo comando `/up` para descargar e instalar la última versión automáticamente
- Nuevo servicio `UpdateService` para gestionar actualizaciones
- Detección automática del sistema operativo y arquitectura (linux/amd64, linux/arm64, windows/amd64)
- Reemplazo seguro del binario en ejecución con backup automático
- Actualización automática del archivo `version.txt`
- Archivo `version.txt` para tracking de versión actual
- Carpeta `build/` para descargas temporales de actualizaciones
- Soporte para compilación cruzada multiplataforma
- Compatibilidad completa con Alpine Linux (ash shell con funciones POSIX)
- Funciones shell con soporte de argumentos (`"$@"`)

### Mejorado
- Sistema de concurrencia con semáforo global funcional (10 slots compartidos)
- Manejo graceful de señales (shutdown limpio con Ctrl+C)
- Sistema de logging con manejo robusto de mensajes largos
- Compatibilidad multiplataforma mejorada (Linux, Windows, macOS, ARM)
- Interfaz de usuario con botones persistentes y emojis descriptivos
- Documentación completa del proyecto (README actualizado)

### Corregido
- **CRÍTICO**: Semáforo de concurrencia ahora funciona correctamente (antes se creaba en cada iteración del bucle)
- **CRÍTICO**: Shadowing de `sigChan` eliminado (causaba fugas de memoria)
- **CRÍTICO**: Panic en `Log.go` por mensajes largos (negative Repeat count)
- **IMPORTANTE**: Error `invalid cross-device link` al reemplazar binarios entre diferentes filesystems
- **IMPORTANTE**: Error `text file busy` al intentar sobrescribir binario en ejecución
- **IMPORTANTE**: Error `no such file or directory` con backups anidados (`.old.old`)
- Funciones shell en ash/POSIX ahora pasan argumentos correctamente

### Cambiado
- `CommandExecutor`: Funciones POSIX ahora usan `"$@"` para pasar argumentos
- `Log.formatearMensaje()`: Truncamiento de líneas largas antes del padding
- `main.go`: Semáforo movido fuera del bucle de procesamiento
- `main.go`: Eliminada redeclaración de `sigChan` dentro del bucle

### Técnico
- Nuevo archivo: `controller/UpdateService.go`
- Nuevo método: `BotHandler.handleUpdate()`
- Nuevos métodos en `UpdateService`: `GetCurrentVersion()`, `ReplaceExecutable()`, `UpdateVersionFile()`
- Nuevo helper: `UpdateService.copyFile()` para copias seguras entre filesystems
- Estrategia de renombramiento para reemplazo de binarios en ejecución

---

## [v1.0.0] - 2026-01-09

### Añadido
- Sistema completo de bot de Telegram
- Ejecución de comandos del sistema con timeout
- Soporte para shell aliases (bash, zsh, ash)
- Sistema de logging con rotación automática
- Información de red (IP local, pública, hostname)
- Control de acceso por lista blanca de usuarios
- Semáforo de concurrencia (máximo 10 comandos simultáneos)
- Botones inline y reply keyboard
- Soporte para Alpine Linux (ash shell)

### Características principales
- `/start` - Información completa del bot y sistema
- `/estado` - Ver IPs y estado de red
- `/comando <cmd>` - Ejecutar comandos del sistema
- `/ayuda` - Lista de comandos disponibles
- Sistema de timeout (30 segundos por defecto)
- Truncado de output largo (4000 caracteres máximo)