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

# Escapar las variables para JSON
ESCAPED_TITLE=$(escape_json "$RELEASE_TITLE")
ESCAPED_NOTES=$(escape_json "$RELEASE_NOTES")

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