# Changelog

Todos los cambios notables en este proyecto serán documentados en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/),
y este proyecto sigue [Semantic Versioning](https://semver.org/lang/es/).

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