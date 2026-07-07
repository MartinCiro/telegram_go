# Guía para configurar bot de telegram

## 📋 Requisitos y configuración inicial

### 1. Configuración de variables de entorno

Crea un archivo `.env` basado en `example` con esta estructura:

```bash

# Configuración bot de telegram
TELEGRAM_TOKEN=Token de bot en telegram
TELEGRAM_CHAT=id usuario
```

### Actualizar dependencias (Solo desarrollo)

```bash
python -m venv venv; venv\Scripts\activate; pip install pipreqs; pipreqs . --force
```

### Ejecutar proyecto

```bash
python main.py
```

## 🛠️ Procesos de automatización

### Conversión de archivo *".py"* a ejecutable *".exe"*

```bash
py -m PyInstaller --icon="ruta-absoluta-archivo-ico" ruta-abosulta-main-proyecto

# Ejemplo:
python -m PyInstaller --icon="vendor/favicon.ico" main.py
```

#### 🚀 Opciones de compilación:

- `--onefile`: Genera un solo archivo ejecutable
- `--windowed`: Ejecución sin ventana de terminal

> **Nota**: Requiere `pip install pyinstaller pillow` y el reemplazo del key en el archivo helpers (linea 11)

🔧 **Herramienta útil**: [Complemento RPA para Firefox](https://addons.mozilla.org/en-US/firefox/addon/rpa/)

---

## 📂 **Estructura del Proyecto**

```

/core
  ├── /controller            # Lógica de negocio
  │   ├── utils              # Metodos reutilizables o compartidos
  ├── /plugins               # Carpeta contenedora de los plugins, librerías o ejecutables
  ├── /vendor                # Contiene archivos temp, imagenes, txt
```

## 🔄 Diagrama de Ejecución

```mermaid
graph TD
    A[Inicio] --> B[Configurar variables]
    B --> C{¿Actualizar dependencias?}
    C -->|Sí| D[Ejecutar pipreqs --force]
    C -->|No| E[Instalar requirements.txt]
    D --> F[Ejecutar scraper]
    E --> F
    F --> G{¿Compilar a .exe?}
    G -->|Sí| H[Usar PyInstaller]
    G -->|No| J[Finalizar]
    H --> I[Generar ejecutable]
    I --> J
```

## 🔄 Diagrama del Flujo del Scraper

```mermaid
graph TD
    A[Inicio] --> C{¿Está logeado?}
    C -->|Sí| D[Capturar No. Post]
    C -->|No| E[Iniciar sesión]
    E --> C
    D --> F{¿Existe en Faiss?}
    F -->|Sí| G[Leer comentarios]
    G --> I{¿Existe estado?}
    I -->|Sí| J[Actualizar]
    J --> K[Siguiente]
    I -->|No| K
    K --> F
    F -->|No| H[Extraer Datos]
    H --> Ha{¿Existe en Faiss?}
    Ha -->|Sí| Hb[No guardar]
    Hb --> K
    Ha -->|No| Hc[Cargar en BD]
    Hc --> K
```

#### 💡 **Creditos**

[Plantilla base](https://github.com/villalbaluis/arquitectura-bots-python) proporcionada por [Luis Villalba](https://github.com/villalbaluis)