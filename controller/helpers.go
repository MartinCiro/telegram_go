package controller

import "os"

// hostname es un wrapper para aislar la dependencia de os
func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "Desconocido"
	}
	return h
}
