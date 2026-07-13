package controller

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// TunnelService gestiona el túnel Cloudflare y la información del sistema
type TunnelService struct {
	Log         *Log
	LogFile     string
	PidFile     string
	UrlFile     string
	MaxWaitTime time.Duration
}

// NewTunnelService crea una nueva instancia del servicio
func NewTunnelService(log *Log) *TunnelService {
	return &TunnelService{
		Log:         log,
		LogFile:     "/tmp/cloudflared.log",
		PidFile:     "/tmp/cloudflared.pid",
		UrlFile:     "/tmp/current_tunnel_url.txt",
		MaxWaitTime: 60 * time.Second,
	}
}

// StartTunnel inicia el túnel de Cloudflare en segundo plano
func (s *TunnelService) StartTunnel() error {
	if s.Log != nil {
		s.Log.Comentario("INFO", "Iniciando túnel Cloudflare...")
	}

	// Detener cualquier instancia previa
	s.StopTunnel()
	time.Sleep(2 * time.Second)

	// Limpiar archivos anteriores
	os.Remove(s.LogFile)
	os.Remove(s.UrlFile)

	// Comando: cloudflared tunnel --url ssh://localhost:22
	cmd := exec.Command("cloudflared", "tunnel", "--url", "ssh://localhost:22")

	// Redirigir salida al archivo de log
	logFile, err := os.OpenFile(s.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("error abriendo archivo de log: %v", err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	// Iniciar en segundo plano
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("error ejecutando cloudflared: %v (¿está instalado?)", err)
	}

	// Guardar PID
	pidStr := strconv.Itoa(cmd.Process.Pid)
	if err := os.WriteFile(s.PidFile, []byte(pidStr), 0644); err != nil {
		if s.Log != nil {
			s.Log.Comentario("WARNING", fmt.Sprintf("No se pudo guardar PID: %v", err))
		}
	}

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Cloudflared iniciado en segundo plano (PID: %s)", pidStr))
	}

	// Esperar a que se genere la URL
	return s.waitForTunnelURL()
}

// waitForTunnelURL espera a que cloudflared escriba la URL en el log
func (s *TunnelService) waitForTunnelURL() error {
	start := time.Now()
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9.-]+\.trycloudflare\.com`)

	for time.Since(start) < s.MaxWaitTime {
		// Leer el archivo de log
		content, err := os.ReadFile(s.LogFile)
		if err == nil {
			match := urlRegex.FindString(string(content))
			if match != "" {
				// Guardar URL en archivo
				os.WriteFile(s.UrlFile, []byte(match), 0644)
				if s.Log != nil {
					s.Log.Comentario("SUCCESS", fmt.Sprintf("Túnel establecido: %s", match))
				}
				return nil
			}
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("timeout: no se obtuvo la URL del túnel en %v", s.MaxWaitTime)
}

// StopTunnel detiene el proceso de cloudflared
func (s *TunnelService) StopTunnel() {
	if s.Log != nil {
		s.Log.Comentario("INFO", "Deteniendo túnel Cloudflare...")
	}

	// Intentar matar por PID primero
	if pidData, err := os.ReadFile(s.PidFile); err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(string(pidData)))
		if pid > 0 {
			process, err := os.FindProcess(pid)
			if err == nil {
				process.Kill()
			}
		}
	}

	// Fallback: matar por nombre de proceso (equivalente a pkill)
	exec.Command("pkill", "-f", "cloudflared.*ssh://localhost:22").Run()

	os.Remove(s.PidFile)
	os.Remove(s.UrlFile)

	if s.Log != nil {
		s.Log.Comentario("INFO", "Túnel detenido")
	}
}

// CheckTunnel verifica el estado del túnel
func (s *TunnelService) CheckTunnel() (string, int, error) {
	// 1. Obtener URL
	url := s.getTunnelURL()
	if url == "" {
		return "Sin URL", 0, fmt.Errorf("no hay URL de túnel disponible")
	}

	// 2. Verificar si el proceso sigue vivo
	if !s.isProcessRunning() {
		if s.Log != nil {
			s.Log.Comentario("WARNING", "Proceso cloudflared no encontrado, intentando reiniciar...")
		}
		if err := s.StartTunnel(); err != nil {
			return "Falló al reiniciar", 0, err
		}
		url = s.getTunnelURL() // Obtener nueva URL si se reinició
	}

	// 3. Verificación HTTP
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(url)

	if err != nil {
		return "Caído (Timeout/Error)", 0, err
	}
	defer resp.Body.Close()

	code := resp.StatusCode
	var status string

	switch code {
	case 200:
		status = "Funcional (HTTP 200)"
	case 530:
		status = "Activo, origen SSH no responde (HTTP 530 - Normal)"
	default:
		status = fmt.Sprintf("No funcional (HTTP %d)", code)
	}

	return status, code, nil
}

// getTunnelURL obtiene la URL del archivo o del log
func (s *TunnelService) getTunnelURL() string {
	// Intentar leer del archivo
	if data, err := os.ReadFile(s.UrlFile); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Fallback: buscar en el log
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9.-]+\.trycloudflare\.com`)
	if content, err := os.ReadFile(s.LogFile); err == nil {
		if match := urlRegex.FindString(string(content)); match != "" {
			os.WriteFile(s.UrlFile, []byte(match), 0644) // Cachear
			return match
		}
	}
	return ""
}

// isProcessRunning verifica si el PID guardado está activo
func (s *TunnelService) isProcessRunning() bool {
	pidData, err := os.ReadFile(s.PidFile)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// En Unix, FindProcess siempre tiene éxito, hay que enviar señal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetSystemInfo retorna información básica del sistema (Temp y RAM)
func (s *TunnelService) GetSystemInfo() string {
	var info strings.Builder
	info.WriteString("📊 *Información del Sistema*\n\n")

	// 1. Temperatura (Solo Linux/Raspberry Pi)
	if runtime.GOOS == "linux" {
		if tempData, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
			tempStr := strings.TrimSpace(string(tempData))
			if tempInt, err := strconv.Atoi(tempStr); err == nil {
				tempC := float64(tempInt) / 1000.0
				info.WriteString(fmt.Sprintf("🌡 *Temperatura:* %.1f °C\n", tempC))
			}
		} else {
			info.WriteString("🌡 *Temperatura:* No disponible\n")
		}
	} else {
		info.WriteString(fmt.Sprintf("🌡 *Temperatura:* No disponible en %s\n", runtime.GOOS))
	}

	// 2. Memoria RAM
	if runtime.GOOS == "linux" {
		cmd := exec.Command("free", "-m")
		if out, err := cmd.Output(); err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "Mem:") {
					fields := strings.Fields(line)
					if len(fields) >= 3 {
						used := fields[2]
						total := fields[1]
						info.WriteString(fmt.Sprintf("💾 *RAM:* %s / %s MB\n", used, total))
					}
					break
				}
			}
		}
	} else {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		info.WriteString(fmt.Sprintf("💾 *RAM Uso Go:* %.2f MB\n", float64(m.Alloc)/1024/1024))
	}

	// 3. Arquitectura
	info.WriteString(fmt.Sprintf("🖥️ *Arquitectura:* %s / %s", runtime.GOOS, runtime.GOARCH))

	return info.String()
}

// GetTunnelURL obtiene la URL del archivo o del log (Pública)
func (s *TunnelService) GetTunnelURL() string {
	// Intentar leer del archivo caché primero
	if data, err := os.ReadFile(s.UrlFile); err == nil {
		url := strings.TrimSpace(string(data))
		if url != "" && s.isProcessRunning() {
			return url // La URL existe Y el proceso sigue vivo
		}
	}

	// Fallback: buscar en el log si el archivo no existe o el proceso murió
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9.-]+\.trycloudflare\.com`)
	if content, err := os.ReadFile(s.LogFile); err == nil {
		if match := urlRegex.FindString(string(content)); match != "" {
			os.WriteFile(s.UrlFile, []byte(match), 0644)
			return match
		}
	}
	return ""
}

// EnsureTunnelRunning verifica si el túnel está activo.
// Si no lo está, lo inicia automáticamente y espera la URL.
func (s *TunnelService) EnsureTunnelRunning() (string, error) {
	// 1. Verificar si ya está corriendo y tenemos la URL
	url := s.GetTunnelURL()
	if url != "" && s.isProcessRunning() {
		if s.Log != nil {
			s.Log.Comentario("INFO", "Túnel ya está activo, usando URL existente")
		}
		return url, nil
	}

	// 2. Si no está activo o no hay URL, iniciarlo automáticamente
	if s.Log != nil {
		s.Log.Comentario("INFO", "Túnel inactivo o sin URL. Iniciando automáticamente...")
	}

	err := s.StartTunnel()
	if err != nil {
		return "", fmt.Errorf("no se pudo iniciar el túnel: %v", err)
	}

	// 3. Obtener la URL recién generada
	newUrl := s.GetTunnelURL()
	if newUrl == "" {
		return "", fmt.Errorf("el túnel inició, pero no se pudo extraer la URL")
	}

	return newUrl, nil
}
