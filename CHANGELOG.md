## [v1.0.4.4] - 2026-07-15

### Añadido
- **Fallback de MAC estática**: Nueva variable `WOL_TARGET_MAC` en `.env` para garantizar el Wake-on-LAN incluso si la caché ARP del sistema está limpia.
- **Túneles dinámicos**: Nuevo comando `/tunnel [protocolo] [puerto]` (ej: `/tunnel http 3000`) con valores por defecto inteligentes (`http` y `8443`) sin hardcodear en `.env`.
- **Limpieza de URL**: Eliminación automática del prefijo `https://` en las respuestas de túnel para una presentación más limpia.

### Mejorado
- **WoL No Bloqueante**: El comando `/wol` ahora responde inmediatamente tras enviar el paquete mágico, liberando la cola del bot. La confirmación de encendido la realiza el bot del equipo destino al iniciar (vía notificación de inicio).
- **Robustez de red**: El servicio WoL intenta primero la resolución dinámica por ARP (`ip neigh` / `arp -n`) y cae elegantemente a la MAC estática configurada si falla.
- **Experiencia de usuario**: Mensajes de "Procesando..." para operaciones que requieren espera, sin bloquear la capacidad del bot para atender otros comandos.

### Corregido
- **CRÍTICO**: Prevención de bloqueos de cola de mensajes durante el tiempo de arranque del equipo (2+ minutos) tras un comando WoL.
- **IMPORTANTE**: Fallos de WoL tras largos periodos de inactividad debido a la expiración de entradas en la tabla ARP.
- **IMPORTANTE**: Dependencia de acortadores de URL externos que fallaban con dominios temporales de Cloudflare.

### Cambiado
- `WolService.go`: Refactorización de `ExecuteWol` para ser asíncrono y no bloqueante.
- `WolService.go`: `getMACAddress` ahora prioriza ARP, pero usa `WOL_TARGET_MAC` como fallback infalible.
- `TunnelService.go`: `StartTunnel` ahora es un alias de `StartTunnelCustom`, permitiendo flexibilidad total de protocolo y puerto.
- `BotHandler.go`: Parser de argumentos para el comando `/tunnel`.

### Técnico
- Uso de `strings.ReplaceAll` y `strings.ToLower` para normalización automática de formatos de dirección MAC (guiones o dos puntos).
- Implementación de `WithPreMessage` en `Response` para feedback inmediato al usuario.
- Gestión de procesos en segundo plano sin depender de herramientas externas como `screen` o `nohup`.

---