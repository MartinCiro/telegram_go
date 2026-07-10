#!/bin/bash

# Permitir pasar el archivo .env como parámetro
ENV_FILE="${1:-.env.compiler}"

# Cargar archivo de configuración
if [ -f "$ENV_FILE" ]; then
    set -a
    source "$ENV_FILE"
    set +a
else
    echo "❌ Error: No se encontró el archivo $ENV_FILE"
    exit 1
fi

# Validar variables requeridas
if [ -z "$GITHUB_TOKEN" ]; then
    echo "❌ Error: GITHUB_TOKEN no está definido en $ENV_FILE"
    exit 1
fi

if [ -z "$GITHUB_REPO" ]; then
    echo "❌ Error: GITHUB_REPO no está definido en $ENV_FILE"
    exit 1
fi

if [ -z "$RELEASE_TAG" ]; then
    echo "❌ Error: RELEASE_TAG no está definido en $ENV_FILE"
    exit 1
fi

if [ -z "$RELEASE_TITLE" ]; then
    echo "❌ Error: RELEASE_TITLE no está definido en $ENV_FILE"
    exit 1
fi

# Función para escapar strings para JSON
escape_json() {
    local str="$1"
    # Escapar backslashes, comillas dobles, y convertir saltos de línea a \n
    str="${str//\\/\\\\}"
    str="${str//\"/\\\"}"
    str=$(echo "$str" | awk '{printf "%s\\n", $0}' | sed 's/\\n$//')
    echo "$str"
}

# Función para compilar binarios
compile_binaries() {
    echo "🔨 Compilando binarios para $RELEASE_TAG..."
    
    # Crear carpeta build si no existe
    mkdir -p build
    
    # Limpiar binarios anteriores si la versión cambió
    local last_version_file="build/.last_version"
    local last_version=""
    
    if [ -f "$last_version_file" ]; then
        last_version=$(cat "$last_version_file")
    fi
    
    if [ "$last_version" != "$RELEASE_TAG" ]; then
        echo "  🗑️  Versión cambió: $last_version → $RELEASE_TAG"
        echo "  🗑️  Limpiando binarios anteriores..."
        rm -f build/bot-telegram-linux-amd64
        rm -f build/bot-telegram-linux-arm64
        rm -f build/bot-telegram-windows-amd64.exe
    fi
    
    # Compilar para Linux amd64
    if [ ! -f "build/bot-telegram-linux-amd64" ]; then
        echo "  → Compilando Linux amd64..."
        GOOS=linux GOARCH=amd64 go build -o build/bot-telegram-linux-amd64 main.go
        if [ $? -eq 0 ]; then
            echo "    ✅ Linux amd64 compilado"
        else
            echo "    ❌ Error compilando Linux amd64"
            exit 1
        fi
    else
        echo "  ✓ Linux amd64 ya existe"
    fi
    
    # Compilar para Linux arm64
    if [ ! -f "build/bot-telegram-linux-arm64" ]; then
        echo "  → Compilando Linux arm64..."
        GOOS=linux GOARCH=arm64 go build -o build/bot-telegram-linux-arm64 main.go
        if [ $? -eq 0 ]; then
            echo "    ✅ Linux arm64 compilado"
        else
            echo "    ❌ Error compilando Linux arm64"
            exit 1
        fi
    else
        echo "  ✓ Linux arm64 ya existe"
    fi
    
    # Compilar para Windows amd64
    if [ ! -f "build/bot-telegram-windows-amd64.exe" ]; then
        echo "  → Compilando Windows amd64..."
        GOOS=windows GOARCH=amd64 go build -o build/bot-telegram-windows-amd64.exe main.go
        if [ $? -eq 0 ]; then
            echo "    ✅ Windows amd64 compilado"
        else
            echo "    ❌ Error compilando Windows amd64"
            exit 1
        fi
    else
        echo "  ✓ Windows amd64 ya existe"
    fi
    
    # Guardar la versión actual
    echo "$RELEASE_TAG" > "$last_version_file"
    
    echo "✅ Todos los binarios están listos para $RELEASE_TAG"
}

# Escapar las variables para JSON
ESCAPED_TITLE=$(escape_json "$RELEASE_TITLE")
ESCAPED_NOTES=$(escape_json "$RELEASE_NOTES")

# Compilar binarios si no existen o si la versión cambió
compile_binaries

# Crear release
echo "🚀 Creando release $RELEASE_TAG en $GITHUB_REPO..."

RESPONSE=$(curl -s -X POST \
  -H "Authorization: token $GITHUB_TOKEN" \
  -H "Accept: application/vnd.github.v3+json" \
  https://api.github.com/repos/$GITHUB_REPO/releases \
  -d "{
    \"tag_name\": \"$RELEASE_TAG\",
    \"name\": \"$ESCAPED_TITLE\",
    \"body\": \"$ESCAPED_NOTES\",
    \"draft\": false,
    \"prerelease\": false
  }")

# Extraer el ID del release
RELEASE_ID=$(echo $RESPONSE | grep -o '"id": *[0-9]*' | head -1 | grep -o '[0-9]*')

if [ -z "$RELEASE_ID" ]; then
    echo "❌ Error creando release:"
    echo $RESPONSE
    exit 1
fi

echo "✅ Release creado con ID: $RELEASE_ID"

# Subir archivos
echo "📤 Subiendo binarios..."
for file in build/bot-telegram-linux-amd64 build/bot-telegram-linux-arm64 build/bot-telegram-windows-amd64.exe; do
    if [ -f "$file" ]; then
        FILENAME=$(basename $file)
        echo "  → Subiendo $FILENAME..."
        curl -s -X POST \
          -H "Authorization: token $GITHUB_TOKEN" \
          -H "Content-Type: application/octet-stream" \
          --data-binary @"$file" \
          "https://uploads.github.com/repos/$GITHUB_REPO/releases/$RELEASE_ID/assets?name=$FILENAME" > /dev/null
        echo "    ✅ $FILENAME subido"
    else
        echo "  ⚠️  $file no encontrado (saltando)"
    fi
done

echo "🎉 Release $RELEASE_TAG completado"
echo "🔗 https://github.com/$GITHUB_REPO/releases/tag/$RELEASE_TAG"