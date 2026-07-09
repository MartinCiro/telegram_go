// controller/UpdateService.go
package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// UpdateService servicio para actualizar el bot desde GitHub
type UpdateService struct {
	RepoURL string
	Timeout time.Duration
	Log     *Log
}

// GitHubRelease estructura de la respuesta de la API de GitHub
type GitHubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Assets  []GitHubAsset `json:"assets"`
}

// GitHubAsset estructura de un asset en GitHub
type GitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateResult resultado de la actualización
type UpdateResult struct {
	CurrentVersion string
	NewVersion     string
	DownloadedFile string
	FileSize       int64
	Replaced       bool
	Err            error
}

// NewUpdateService crea un nuevo servicio de actualización
func NewUpdateService(log *Log) *UpdateService {
	return &UpdateService{
		RepoURL: "https://api.github.com/repos/MartinCiro/telegram_go/releases/latest",
		Timeout: 30 * time.Second,
		Log:     log,
	}
}

// CheckAndUpdate verifica si hay una nueva versión y la descarga
func (s *UpdateService) CheckAndUpdate() *UpdateResult {
	result := &UpdateResult{}

	// 1. Obtener versión actual
	currentVersion, err := s.GetCurrentVersion()
	if err != nil {
		result.Err = fmt.Errorf("error leyendo versión actual: %v", err)
		if s.Log != nil {
			s.Log.Error(result.Err.Error(), "UpdateService")
		}
		return result
	}
	result.CurrentVersion = currentVersion

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Versión actual: %s", currentVersion))
	}

	// 2. Obtener la última release de GitHub
	release, err := s.getLatestRelease()
	if err != nil {
		result.Err = fmt.Errorf("error obteniendo última release: %v", err)
		if s.Log != nil {
			s.Log.Error(result.Err.Error(), "UpdateService")
		}
		return result
	}

	result.NewVersion = release.TagName

	// 3. Comparar versiones
	if currentVersion == release.TagName {
		result.Err = fmt.Errorf("ya tienes la última versión (%s)", currentVersion)
		if s.Log != nil {
			s.Log.Comentario("INFO", "No hay actualizaciones disponibles")
		}
		return result
	}

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Nueva versión disponible: %s -> %s", currentVersion, release.TagName))
	}

	// 4. Detectar el sistema actual
	osName := runtime.GOOS
	archName := runtime.GOARCH

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Sistema detectado: %s/%s", osName, archName))
	}

	// 5. Buscar el asset correcto para este sistema
	assetName := s.getAssetName(osName, archName)
	var targetAsset *GitHubAsset

	for i := range release.Assets {
		if release.Assets[i].Name == assetName {
			targetAsset = &release.Assets[i]
			break
		}
	}

	if targetAsset == nil {
		result.Err = fmt.Errorf("no se encontró un binario para %s/%s (buscando: %s)", osName, archName, assetName)
		if s.Log != nil {
			s.Log.Error(result.Err.Error(), "UpdateService")
		}
		return result
	}

	// 6. Descargar el binario
	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Descargando %s (%.2f MB)...", targetAsset.Name, float64(targetAsset.Size)/1024/1024))
	}

	downloadedFile, fileSize, err := s.downloadAsset(targetAsset)
	if err != nil {
		result.Err = fmt.Errorf("error descargando binario: %v", err)
		if s.Log != nil {
			s.Log.Error(result.Err.Error(), "UpdateService")
		}
		return result
	}

	result.DownloadedFile = downloadedFile
	result.FileSize = fileSize

	if s.Log != nil {
		s.Log.Comentario("SUCCESS", fmt.Sprintf("Descarga completada: %s", downloadedFile))
	}

	// 7. Reemplazar el binario actual
	if err := s.ReplaceExecutable(downloadedFile); err != nil {
		result.Err = fmt.Errorf("error reemplazando binario: %v", err)
		if s.Log != nil {
			s.Log.Error(result.Err.Error(), "UpdateService")
		}
		return result
	}

	// 8. Actualizar version.txt
	if err := s.UpdateVersionFile(release.TagName); err != nil {
		result.Err = fmt.Errorf("error actualizando version.txt: %v", err)
		if s.Log != nil {
			s.Log.Error(result.Err.Error(), "UpdateService")
		}
		return result
	}

	result.Replaced = true // ← Nuevo campo en UpdateResult

	if s.Log != nil {
		s.Log.Comentario("SUCCESS", "Actualización completada exitosamente")
	}

	return result
}

// getLatestRelease consulta la API de GitHub
func (s *UpdateService) getLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: s.Timeout}

	req, err := http.NewRequest("GET", s.RepoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "TelegramBot-Updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API respondió con status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, err
	}

	return &release, nil
}

// getAssetName retorna el nombre del archivo según el sistema
func (s *UpdateService) getAssetName(osName, archName string) string {
	var osPart, archPart string

	switch osName {
	case "linux":
		osPart = "linux"
	case "windows":
		osPart = "windows"
	case "darwin":
		osPart = "darwin"
	default:
		osPart = osName
	}

	switch archName {
	case "amd64":
		archPart = "amd64"
	case "arm64":
		archPart = "arm64"
	case "arm":
		archPart = "arm"
	default:
		archPart = archName
	}

	filename := fmt.Sprintf("bot-telegram-%s-%s", osPart, archPart)
	if osName == "windows" {
		filename += ".exe"
	}

	return filename
}

// downloadAsset descarga el archivo y lo guarda
func (s *UpdateService) downloadAsset(asset *GitHubAsset) (string, int64, error) {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(asset.BrowserDownloadURL)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("error descargando: status %d", resp.StatusCode)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", 0, err
	}

	buildDir := filepath.Join(cwd, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", 0, fmt.Errorf("error creando carpeta build: %v", err)
	}

	filePath := filepath.Join(buildDir, asset.Name)

	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	written, err := io.Copy(file, resp.Body)
	if err != nil {
		return "", 0, err
	}

	// Hacer ejecutable en Unix (✅ Corregido: 0o755 en lugar de 0755)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(filePath, 0o755); err != nil {
			return "", 0, err
		}
	}

	return filePath, written, nil
}

// FormatForTelegram formatea el resultado para Telegram
func (r *UpdateResult) FormatForTelegram() string {
	var sb strings.Builder

	sb.WriteString("🔄 *Actualización del Bot*\n\n")

	if r.Err != nil {
		if strings.Contains(r.Err.Error(), "ya tienes la última versión") {
			sb.WriteString(fmt.Sprintf("✅ *Ya estás en la última versión:* %s\n", r.CurrentVersion))
			sb.WriteString("\n💡 No se requiere actualización.")
			return sb.String()
		}
		sb.WriteString(fmt.Sprintf("❌ *Error:* %s", r.Err.Error()))
		return sb.String()
	}

	sb.WriteString(fmt.Sprintf("📌 *Versión anterior:* %s\n", r.CurrentVersion))
	sb.WriteString(fmt.Sprintf("✅ *Nueva versión:* %s\n\n", r.NewVersion))
	sb.WriteString(fmt.Sprintf("💾 *Tamaño:* %.2f MB\n\n", float64(r.FileSize)/1024/1024))

	if r.Replaced {
		sb.WriteString("✅ *Binario reemplazado correctamente*\n")
		sb.WriteString("✅ *version.txt actualizado*\n\n")
		sb.WriteString("💡 *Próximo paso:*\n")
		sb.WriteString("Reinicia el bot para usar la nueva versión.")
	} else {
		sb.WriteString(fmt.Sprintf("📁 *Ubicación:* `%s`\n\n", r.DownloadedFile))
		sb.WriteString("💡 *Próximos pasos:*\n")
		sb.WriteString("1. Detén el bot actual (Ctrl+C)\n")
		sb.WriteString("2. Ejecuta el nuevo binario descargado\n")
	}

	return sb.String()
}

func (s *UpdateService) GetCurrentVersion() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	versionFile := filepath.Join(cwd, "version.txt")
	data, err := os.ReadFile(versionFile)
	if err != nil {
		// Si no existe, asumir v0.0.0
		if os.IsNotExist(err) {
			return "v0.0.0", nil
		}
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func (s *UpdateService) GetExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return exe, nil
}

// ReplaceExecutable reemplaza el binario actual con el nuevo
func (s *UpdateService) ReplaceExecutable(newBinaryPath string) error {
	currentExe, err := s.GetExecutablePath()
	if err != nil {
		return fmt.Errorf("error obteniendo ruta del ejecutable actual: %v", err)
	}

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Ejecutable actual: %s", currentExe))
		s.Log.Comentario("INFO", fmt.Sprintf("Nuevo binario: %s", newBinaryPath))
	}

	// ← FIX: Detectar si el ejecutable actual ya es un backup (.old)
	isAlreadyBackup := strings.HasSuffix(currentExe, ".old")

	if isAlreadyBackup {
		if s.Log != nil {
			s.Log.Comentario("WARNING", "El ejecutable actual ya es un backup (.old)")
			s.Log.Comentario("INFO", "Copiando directamente sin crear otro backup")
		}

		// Simplemente copiar sobre el archivo existente
		return s.copyFile(newBinaryPath, currentExe)
	}

	// Estrategia normal: renombrar actual a .old y crear nuevo
	backupPath := currentExe + ".old"

	if err := os.Rename(currentExe, backupPath); err != nil {
		return fmt.Errorf("error creando backup del binario actual: %v", err)
	}

	if s.Log != nil {
		s.Log.Comentario("INFO", fmt.Sprintf("Backup creado: %s", backupPath))
	}

	// Copiar el nuevo binario
	if err := s.copyFile(newBinaryPath, currentExe); err != nil {
		// Rollback: restaurar backup
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("error copiando nuevo binario: %v", err)
	}

	// Intentar eliminar el backup (fallará si está en uso, pero eso está bien)
	if err := os.Remove(backupPath); err != nil {
		if s.Log != nil {
			s.Log.Comentario("INFO", fmt.Sprintf("Backup %s se eliminará al reiniciar", backupPath))
		}
	}

	// Eliminar archivo descargado
	os.Remove(newBinaryPath)

	if s.Log != nil {
		s.Log.Comentario("SUCCESS", "Binario reemplazado correctamente")
	}

	return nil
}

// copyFile copia un archivo de origen a destino
func (s *UpdateService) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return os.Chmod(dst, 0755)
}

// UpdateVersionFile actualiza el archivo version.txt con la nueva versión
func (s *UpdateService) UpdateVersionFile(newVersion string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	versionFile := filepath.Join(cwd, "version.txt")
	if err := os.WriteFile(versionFile, []byte(newVersion+"\n"), 0644); err != nil {
		return fmt.Errorf("error actualizando version.txt: %v", err)
	}

	if s.Log != nil {
		s.Log.Comentario("SUCCESS", fmt.Sprintf("version.txt actualizado a %s", newVersion))
	}

	return nil
}
