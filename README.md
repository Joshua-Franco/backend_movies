# MovieVerse - Backend API

Backend en Go para la plataforma de streaming MovieVerse.

## Stack

- **Lenguaje:** Go 1.26
- **Router:** chi/v5
- **Base de datos:** PostgreSQL 18 (pgx/v5)
- **Config:** godotenv
- **Logs:** fatih/color

## Inicio rápido

```bash
# Instalar dependencias
go mod tidy

# Configurar variables de entorno
copy .env.example .env
# Editar .env con tus credenciales

# Compilar
go build -o movienova.exe .

# Ejecutar
./movienova.exe
```

Backend corre en `http://localhost:8080`.

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

## Estructura

```
backend_movies/
├── main.go                     # Entry point: config, DB, migrations, server
├── go.mod                      # Módulo Go y dependencias
├── go.sum                      # Lock de dependencias
├── .env                        # Variables de entorno (no commitear)
├── .gitignore
├── convert.bat                 # Conversor HLS (batch)
├── convert.ps1                 # Conversor HLS (PowerShell, barra de progreso)
├── movienova.exe               # Binario compilado
├── server.log                  # Log del servidor
├── server_err.log              # Log de errores
├── README.md
│
├── config/
│   └── config.go               # Load(): lee .env, valida campos, retorna Config struct
│
├── database/
│   └── postgres.go             # Connect(), RunMigrations(), ScanMediaFiles(), GetStreamInfo()
│                               #   - 4 tablas: tmdb_cache, user_lists, continue_watching, media_files
│                               #   - Scan escanea storage/peliculas/ y registra HLS en media_files
│
├── handlers/
│   ├── trending.go             # GET /api/trending/{mediaType}
│   ├── popular.go              # GET /api/popular/{mediaType}, /top-rated, /now-playing
│   ├── detail.go               # GET /api/detail/{mediaType}/{id} (con videos, credits, similar)
│   ├── search.go               # GET /api/search?q=... (multi: movies, tv, persons)
│   ├── genres.go               # GET /api/genres/{mediaType}, /discover/movie
│   ├── discover.go             # GET /api/discover/companies, /discover/tv-companies
│   ├── person.go               # GET /api/person/{personId} (con combined_credits)
│   ├── stream.go               # GET /api/stream/{tmdbId}, /hls/.../playlist.m3u8, /hls/.../{seg}.ts
│   │                           #   - ServeSegment soporta Range requests (HTTP 206)
│   │                           #   - GET /download/{tmdbId} sirve archivo original
│   └── user.go                 # CRUD: /api/my-list, /api/continue-watching
│                               #   - Upsert con ON CONFLICT
│
├── routes/
│   └── routes.go               # SetupRoutes(): registra todas las rutas en chi.Router
│                               #   - /api/* para endpoints de datos
│                               #   - /hls/* y /download/* fuera de /api (CORS directo)
│
├── middleware/
│   ├── cors.go                 # CORS: Allow-Origin *, expose Content-Range/Content-Length
│   └── logger.go               # Logger de requests con method, path, status, duration
│
├── services/
│   ├── tmdb.go                 # TMDBClient: GetTrending, GetPopular, GetDetail, GetPerson, etc.
│   │                           #   - Cachea respuestas en tmdb_cache (1h TTL)
│   └── cache.go                # CacheService: Set/Get/CleanExpired con sync.RWMutex
│
├── models/
│   ├── movie.go                # MovieInfo, QualityInfo structs
│   └── user.go                 # UserListItem, ContinueWatchingItem, AddToListRequest, etc.
│
├── logger/
│   └── logger.go               # Log colorizado: Info, Success, Warn, Error, Fatal, Banner
│
└── storage/
    └── peliculas/
        └── {movie-name}/
            ├── info.json               # { tmdb_id, title }
            ├── archivo_original/       # Archivo MKV/MP4 original
            │   └── *.mkv
            ├── 1080p/
            │   ├── playlist.m3u8       # HLS master playlist
            │   ├── playlist0.ts        # Segmentos de video
            │   ├── playlist0.vtt       # Subtítulos WebVTT
            │   └── ...
            ├── 720p/
            │   └── ...
            └── 480p/
                └── ...
```

## Endpoints

### Películas y Series

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/trending/{mediaType}` | En tendencia (movie/tv) |
| GET | `/api/popular/{mediaType}?page=N` | Populares con paginación |
| GET | `/api/top-rated/{mediaType}?page=N` | Mejor valoradas |
| GET | `/api/now-playing/{mediaType}?page=N` | En cartelera |
| GET | `/api/detail/{mediaType}/{id}` | Detalle (videos, credits, similar) |
| GET | `/api/genres/{mediaType}` | Lista de géneros |
| GET | `/api/discover/movie?genreId=N&page=N` | Descubrir por género |
| GET | `/api/discover/companies?page=N` | Películas por productoras |
| GET | `/api/discover/tv-companies?page=N` | Series por productoras |

### Búsqueda y Personas

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/search?q=...&page=N` | Búsqueda multi (movies, tv, persons) |
| GET | `/api/person/{id}` | Detalle de persona + filmografía |

### Streaming HLS

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/stream/{tmdbId}` | Info de stream (qualities disponibles) |
| POST | `/api/stream/scan` | Re-escanear archivos locales |
| GET | `/hls/{movie}/{quality}/playlist.m3u8` | Playlist HLS |
| GET | `/hls/{movie}/{quality}/{segment}.ts` | Segmento con Range support |
| GET | `/download/{tmdbId}` | Descargar archivo original |

### Listas de Usuario

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/my-list` | Obtener mi lista |
| POST | `/api/my-list` | Agregar (upsert) |
| DELETE | `/api/my-list/{tmdbId}` | Eliminar |

### Continue Watching

| Método | Ruta | Descripción |
|--------|------|-------------|
| GET | `/api/continue-watching` | Obtener progreso |
| POST | `/api/continue-watching` | Guardar/actualizar progreso (upsert) |
| DELETE | `/api/continue-watching/{tmdbId}` | Eliminar |

## Base de datos

4 tablas PostgreSQL:

| Tabla | Descripción |
|-------|-------------|
| `tmdb_cache` | Cache de respuestas TMDB (JSONB, TTL 1h) |
| `user_lists` | Lista de usuarios (UNIQUE: tmdb_id + media_type) |
| `continue_watching` | Progreso de reproducción (UNIQUE: tmdb_id + media_type) |
| `media_files` | Archivos HLS locales (UNIQUE: tmdb_id + quality) |

## Conversión a HLS

```bash
# PowerShell (con barra de progreso)
convert.ps1 "nombre-pelicula" "ruta/al/archivo.mkv"

# Batch
convert.bat "nombre-pelicula" "ruta/al/archivo.mkv"
```

Crea la estructura en `storage/peliculas/{nombre}/` con `info.json`, `archivo_original/`, `1080p/`, `720p/`, `480p/`.
