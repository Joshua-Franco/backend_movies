# MOVIE NOVA - Backend API

Backend en Go para la plataforma de streaming MOVIE NOVA.

## Dependencias

- Go 1.26+
- PostgreSQL 18
- ffmpeg (para conversión HLS)
- TMDB API key

## Inicio rápido

```bash
# 1. Instalar dependencias
go mod tidy

# 2. Configurar variables de entorno
copy .env.example .env
# Editar .env con tus credenciales

# 3. Ejecutar
go run main.go
```

## Variables de entorno

| Variable | Descripción | Default |
|---|---|---|
| `TMDB_API_KEY` | API key de TMDB | requerido |
| `TMDB_BASE_URL` | URL base de TMDB API | `https://api.themoviedb.org/3` |
| `TMDB_IMAGE_URL` | URL base de imágenes | `https://image.tmdb.org/t/p` |
| `DB_HOST` | Host de PostgreSQL | `localhost` |
| `DB_PORT` | Puerto de PostgreSQL | `5432` |
| `DB_USER` | Usuario de PostgreSQL | `postgres` |
| `DB_PASSWORD` | Contraseña de PostgreSQL | requerido |
| `DB_NAME` | Nombre de la base de datos | `movienova` |
| `DB_SSLMODE` | Modo SSL | `disable` |
| `SERVER_PORT` | Puerto del servidor | `8080` |

## Endpoints

### Películas y Series
- `GET /api/trending` - En tendencia
- `GET /api/popular` - Populares
- `GET /api/top-rated` - Mejor valoradas
- `GET /api/now-playing` - En cartelera
- `GET /api/detail/{mediaType}/{id}` - Detalle con créditos, trailers y similares
- `GET /api/discover/{mediaType}` - Descubrir con filtros
- `GET /api/genres/{mediaType}` - Géneros disponibles

### Búsqueda
- `GET /api/search?q={query}` - Búsqueda multi (películas, series, personas)

### Personas
- `GET /api/person/{id}` - Detalle de persona con filmografía

### Streaming
- `GET /api/stream/{tmdbId}` - Info de stream disponible
- `POST /api/stream/scan` - Escanear archivos locales

### HLS (fuera de /api)
- `GET /hls/{movieName}/{quality}/playlist.m3u8` - Playlist HLS
- `GET /hls/{movieName}/{quality}/{segment}.ts` - Segmento TS

### Descarga
- `GET /download/{tmdbId}` - Descargar archivo original

### Listas de usuario
- `GET /api/my-list` - Obtener mi lista
- `POST /api/my-list` - Agregar a mi lista
- `DELETE /api/my-list/{tmdbId}` - Eliminar de mi lista

### Continue watching
- `GET /api/continue-watching` - Obtener progreso
- `POST /api/continue-watching` - Guardar progreso
- `DELETE /api/continue-watching/{tmdbId}` - Eliminar progreso

## Estructura de almacenamiento

```
storage/peliculas/
  {movie-name}/
    info.json           # Metadata de la película
    archivo_original/   # Archivo original (MKV/MP4)
    1080p/              # HLS 1080p
      playlist.m3u8
      *.ts
    720p/               # HLS 720p
    480p/               # HLS 480p
```

## Conversión a HLS

Usa `convert.bat` o `convert.ps1` para convertir archivos MKV/MP4 a HLS:

```bash
convert.bat "nombre-pelicula" "ruta/al/archivo.mkv"
```
