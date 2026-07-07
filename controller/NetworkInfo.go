package controller

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// NetworkInfo agrupa toda la información de red del sistema
type NetworkInfo struct {
	IPLocal   string
	IPPublica string
	Red       string
	Hostname  string
}

// NetworkService servicio para obtener información de red
type NetworkService struct {
	// URL del servicio para obtener IP pública (configurable)
	IPPublicaURL string
	// Timeout para peticiones HTTP
	Timeout time.Duration
}

// NewNetworkService crea un nuevo servicio con valores por defecto
func NewNetworkService() *NetworkService {
	return &NetworkService{
		IPPublicaURL: "https://api.ipify.org?format=text",
		Timeout:      5 * time.Second,
	}
}

// ObtenerInfo retorna toda la información de red disponible
func (s *NetworkService) ObtenerInfo() NetworkInfo {
	return NetworkInfo{
		IPLocal:   s.obtenerIPLocal(),
		IPPublica: s.obtenerIPPublica(),
		Red:       s.obtenerRed(),
		Hostname:  s.obtenerHostname(),
	}
}

// obtenerIPLocal retorna la primera IP local no-loopback IPv4
func (s *NetworkService) obtenerIPLocal() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Error"
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipnet.IP.IsLoopback() {
			continue
		}
		if ip := ipnet.IP.To4(); ip != nil {
			return ip.String()
		}
	}
	return "No disponible"
}

// obtenerRed retorna la red local en formato CIDR
func (s *NetworkService) obtenerRed() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Error"
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ipnet.IP.IsLoopback() {
			continue
		}
		if ipnet.IP.To4() != nil {
			network := ipnet.IP.Mask(ipnet.Mask)
			ones, _ := ipnet.Mask.Size()
			return fmt.Sprintf("%s/%d", network.String(), ones)
		}
	}
	return "No disponible"
}

// obtenerIPPublica consulta un servicio externo para la IP pública
func (s *NetworkService) obtenerIPPublica() string {
	client := &http.Client{Timeout: s.Timeout}
	ctx, cancel := context.WithTimeout(context.Background(), s.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.IPPublicaURL, nil)
	if err != nil {
		return "Error"
	}

	resp, err := client.Do(req)
	if err != nil {
		return "Error"
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil {
		return "Error"
	}

	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return "Respuesta inválida"
	}
	return ip
}

// obtenerHostname retorna el nombre del host
func (s *NetworkService) obtenerHostname() string {
	return hostname()
}

// FormatearParaTelegram retorna la info formateada en Markdown
func (info *NetworkInfo) FormatearParaTelegram() string {
	return fmt.Sprintf(
		"🌐 *IP Pública:* `%s`\n"+
			"🏠 *IP Local:* `%s`\n"+
			"📡 *Red:* `%s`\n"+
			"💻 *Host:* `%s`",
		info.IPPublica, info.IPLocal, info.Red, info.Hostname,
	)
}
