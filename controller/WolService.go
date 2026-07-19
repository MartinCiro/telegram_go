package controller

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// WolService gestiona el Wake-on-LAN avanzado
type WolService struct {
	TargetIP   string
	TargetUser string
	SSHPort    int
	TargetMAC  string
	Log        *Log
}

// NewWolService crea una nueva instancia del servicio WoL usando la configuración
func NewWolService(config *Config) *WolService {
	return &WolService{
		TargetIP:   config.WolTargetIP,
		TargetUser: config.WolTargetUser,
		SSHPort:    config.WolSSHPort,
		TargetMAC:  config.WolTargetMAC,
		Log:        config.Log,
	}
}

// ExecuteWol ejecuta el flujo completo: verificar -> obtener MAC -> enviar -> verificar
func (s *WolService) ExecuteWol() (string, error) {
	// 1. Verificar si ya está despierto
	if s.isHostAwake() {
		return fmt.Sprintf("✅ El equipo (`%s`) **ya está conectado** y responde. No es necesario enviar WoL.", s.TargetIP), nil
	}

	if s.Log != nil {
		s.Log.Comentario("INFO", "Equipo dormido. Iniciando proceso WoL...")
	}

	// 2. Obtener dirección MAC
	mac, err := s.getMACAddress()
	if err != nil {
		return "", fmt.Errorf("no se pudo obtener la dirección MAC de %s. Detalle: %v", s.TargetIP, err)
	}

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("MAC encontrada: %s. Enviando paquete mágico...", mac))
	}

	// 3. Enviar Paquete Mágico
	if err := s.sendMagicPacket(mac); err != nil {
		return "", fmt.Errorf("error enviando paquete mágico: %v", err)
	}

	if s.Log != nil {
		s.Log.Comentario("SUCCESS", "Paquete WoL enviado exitosamente")
	}

	// 4. Retornar INMEDIATAMENTE sin bloquear la cola del bot (sin time.Sleep)
	// La confirmación real la dará el bot de la máquina destino al iniciarse (paso 6 de main.go)
	successMsg := fmt.Sprintf(
		"✅ *¡Paquete WoL enviado!*\n\n"+
			"El equipo se está iniciando. Este proceso puede tomar un par de minutos.\n\n"+
			"💡 *Recibirás una notificación automática en el chat* cuando el sistema esté completamente operativo y el bot se haya iniciado.",
		s.TargetIP,
	)

	return successMsg, nil
}

// isHostAwake verifica si el puerto SSH del equipo está abierto
func (s *WolService) isHostAwake() bool {
	address := fmt.Sprintf("%s:%d", s.TargetIP, s.SSHPort)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// getMACAddress obtiene la dirección MAC de la IP usando la tabla ARP del sistema
func (s *WolService) getMACAddress() (string, error) {
	// 1. Intentamos con 'ip neigh' (moderno, disponible en Alpine/Linux)
	cmd := exec.Command("ip", "neigh", "show", s.TargetIP)
	output, err := cmd.Output()

	if err != nil {
		// 2. Fallback a 'arp -n' (más antiguo, pero universal)
		cmd = exec.Command("arp", "-n", s.TargetIP)
		output, err = cmd.Output()
		if err != nil {
			// Si falla, limpiamos el output para que el regex falle limpiamente
			// y pasemos al fallback de configuración
			output = []byte("")
		}
	}

	// 3. Expresión regular para encontrar una dirección MAC
	macRegex := regexp.MustCompile(`([0-9a-fA-F]{2}[:-]){5}([0-9a-fA-F]{2})`)
	match := macRegex.FindString(string(output))

	if match != "" {
		// Éxito: Normalizar a formato con dos puntos y minúsculas
		match = strings.ReplaceAll(match, "-", ":")
		return strings.ToLower(match), nil
	}

	// 4. FALLBACK CRÍTICO: Si no se encontró en ARP, usar la MAC configurada en .env
	if s.TargetMAC != "" {
		if s.Log != nil {
			s.Log.Comentario("WARNING", fmt.Sprintf("No se encontró la MAC en la tabla ARP para %s. Usando MAC de respaldo configurada en .env: %s", s.TargetIP, s.TargetMAC))
		}
		// Normalizar también la MAC configurada por si el usuario la puso con guiones
		cleanMAC := strings.ReplaceAll(s.TargetMAC, "-", ":")
		return strings.ToLower(cleanMAC), nil
	}

	// 5. Si no hay ARP ni fallback configurado, retornar error
	return "", fmt.Errorf("no se encontró la MAC en la tabla ARP y no hay una WOL_TARGET_MAC configurada en .env. Intenta hacer ping a %s manualmente primero", s.TargetIP)
}

// sendMagicPacket construye y envía el paquete WoL por broadcast UDP
func (s *WolService) sendMagicPacket(mac string) error {
	macBytes, err := net.ParseMAC(mac)
	if err != nil {
		return fmt.Errorf("MAC inválida: %v", err)
	}

	// El paquete mágico: 6 bytes de 0xFF + 16 repeticiones de la MAC (102 bytes)
	payload := make([]byte, 0, 102)
	for i := 0; i < 6; i++ {
		payload = append(payload, 0xff)
	}
	for i := 0; i < 16; i++ {
		payload = append(payload, macBytes...)
	}

	// Enviar por broadcast
	conn, err := net.Dial("udp", "255.255.255.255:9")
	if err != nil {
		conn, err = net.Dial("udp", "192.168.1.255:9") // Fallback a broadcast local
		if err != nil {
			return fmt.Errorf("no se pudo crear conexión UDP: %v", err)
		}
	}
	defer conn.Close()

	_, err = conn.Write(payload)
	return err
}
