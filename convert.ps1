param(
    [string]$FilePath = ""
)

$ErrorActionPreference = "Stop"

$FFMPEG = "C:\Users\joshu\Downloads\ffmpeg-2026-08-03-git-01a25f74cc-full_build\ffmpeg-2026-08-03-git-01a25f74cc-full_build\bin\ffmpeg.exe"
$FFPROBE = $FFMPEG -replace "ffmpeg\.exe", "ffprobe.exe"
$STORAGE = Join-Path $PSScriptRoot "storage\peliculas"

Write-Host ""
Write-Host "========================================"
Write-Host "   Conversor de Peliculas a HLS"
Write-Host "========================================"
Write-Host ""

if (-not (Test-Path $FFMPEG)) {
    Write-Host "ERROR: ffmpeg no encontrado en:" -ForegroundColor Red
    Write-Host $FFMPEG
    Read-Host "Presiona Enter para salir"
    exit 1
}

if (-not (Test-Path $FFPROBE)) {
    Write-Host "ERROR: ffprobe no encontrado en:" -ForegroundColor Red
    Write-Host $FFPROBE
    Read-Host "Presiona Enter para salir"
    exit 1
}

if (-not $FilePath) {
    $FilePath = Read-Host "Ruta del archivo (mp4/mkv)"
}
$FilePath = $FilePath.Trim('"').Trim("'")

if (-not (Test-Path $FilePath)) {
    Write-Host "ERROR: Archivo no encontrado." -ForegroundColor Red
    Read-Host "Presiona Enter para salir"
    exit 1
}

$fileName = [System.IO.Path]::GetFileNameWithoutExtension($FilePath)
$movieName = $fileName -replace '[\. ]', '-' -replace '_', '-'

Write-Host ""
Write-Host "Obteniendo duracion del video..." -ForegroundColor Cyan

$probeArgs = @(
    "-v", "error",
    "-show_entries", "format=duration",
    "-of", "default=noprint_wrappers=1:nokey=1",
    $FilePath
)
$totalDuration = & $FFPROBE @probeArgs 2>$null
$totalDuration = [double]$totalDuration

if ($totalDuration -le 0) {
    Write-Host "ERROR: No se pudo obtener la duracion del video." -ForegroundColor Red
    Read-Host "Presiona Enter para salir"
    exit 1
}

$totalHours = [math]::Floor($totalDuration / 3600)
$totalMinutes = [math]::Floor(($totalDuration % 3600) / 60)
$totalSeconds = [math]::Floor($totalDuration % 60)
Write-Host "Duracion: $($totalHours)h $($totalMinutes)m $($totalSeconds)s" -ForegroundColor Green

Write-Host ""
Write-Host "Buscando `"$movieName`" en TMDB..." -ForegroundColor Cyan

$apiKey = ""
$envFile = Join-Path $PSScriptRoot ".env"
if (Test-Path $envFile) {
    $envContent = Get-Content $envFile
    foreach ($line in $envContent) {
        if ($line -match "TMDB_API_KEY=(.+)") {
            $apiKey = $Matches[1].Trim()
        }
    }
}

if (-not $apiKey) {
    Write-Host "ERROR: TMDB_API_KEY no encontrada en .env" -ForegroundColor Red
    Read-Host "Presiona Enter para salir"
    exit 1
}

$searchUrl = "https://api.themoviedb.org/3/search/movie?query=$([Uri]::EscapeDataString($movieName))&api_key=$apiKey&language=en-US"

try {
    $response = Invoke-RestMethod -Uri $searchUrl
    $movie = $response.results | Select-Object -First 1
} catch {
    Write-Host "ERROR: Fallo la busqueda en TMDB: $_" -ForegroundColor Red
    Read-Host "Presiona Enter para salir"
    exit 1
}

if (-not $movie) {
    Write-Host "No se encontro la pelicula en TMDB." -ForegroundColor Red
    Read-Host "Presiona Enter para salir"
    exit 1
}

$year = if ($movie.release_date) { $movie.release_date.Substring(0, 4) } else { "N/A" }
Write-Host ""
Write-Host "Encontrado: $($movie.title) ($year) [TMDB ID: $($movie.id)]" -ForegroundColor Green

$confirm = Read-Host "Correcto? (s/n)"
if ($confirm -ne "s") {
    Write-Host "Cancelado." -ForegroundColor Yellow
    exit 0
}

$tmdbId = $movie.id
$title = $movie.title
$outDir = Join-Path $STORAGE $movieName

New-Item -ItemType Directory -Force -Path (Join-Path $outDir "1080p") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $outDir "720p") | Out-Null
New-Item -ItemType Directory -Force -Path (Join-Path $outDir "480p") | Out-Null

@{ tmdb_id = $tmdbId; title = $title } | ConvertTo-Json | Set-Content (Join-Path $outDir "info.json")

Write-Host ""
Write-Host "Guardado en: $outDir" -ForegroundColor Green
Write-Host ""

function Convert-Video {
    param(
        [string]$Quality,
        [string]$Scale,
        [string]$VideoBitrate,
        [string]$AudioBitrate
    )

    $playlistPath = Join-Path $outDir "$Quality\playlist.m3u8"

    Write-Host "============================================" -ForegroundColor White
    Write-Host "  Convirtiendo a $Quality" -ForegroundColor Yellow
    Write-Host "============================================" -ForegroundColor White

    $psi = New-Object System.Diagnostics.ProcessStartInfo
    $psi.FileName = $FFMPEG
    $psi.Arguments = "-i `"$FilePath`" -vf `"$Scale`" -c:v libx264 -preset fast -b:v $VideoBitrate -c:a aac -b:a $AudioBitrate -hls_time 10 -hls_list_size 0 -f hls `"$playlistPath`""
    $psi.UseShellExecute = $false
    $psi.RedirectStandardError = $true
    $psi.RedirectStandardOutput = $true
    $psi.CreateNoWindow = $true

    $process = [System.Diagnostics.Process]::Start($psi)

    $lastPercent = -1

    while (-not $process.HasExited) {
        $line = $process.StandardError.ReadLine()

        if ($line -match "time=(\d{2}):(\d{2}):(\d{2})") {
            $h = [int]$Matches[1]
            $m = [int]$Matches[2]
            $s = [int]$Matches[3]
            $currentSeconds = $h * 3600 + $m * 60 + $s
            $percent = [math]::Min(100, [math]::Floor(($currentSeconds / $totalDuration) * 100))

            if ($percent -ne $lastPercent) {
                $lastPercent = $percent
                $filled = [math]::Floor($percent / 2)
                $empty = 50 - $filled
                $bar = ("#" * $filled) + ("-" * $empty)

                $currentTime = "{0:D2}:{1:D2}:{2:D2}" -f $h, $m, $s
                $totalTime = "{0:D2}:{1:D2}:{2:D2}" -f [math]::Floor($totalDuration/3600), [math]::Floor(($totalDuration%3600)/60), [math]::Floor($totalDuration%60)

                Write-Host "`r[$bar] $percent% ($currentTime / $totalTime)" -NoNewline -ForegroundColor Cyan
            }
        }

        if ($line -match "frame=" -and -not $line -match "time=") {
            # Speed/fps line, skip
        }
    }

    $process.WaitForExit()

    Write-Host ""
    if ($process.ExitCode -eq 0) {
        Write-Host "  $Quality completado!" -ForegroundColor Green
    } else {
        Write-Host "  $Quality fallo (exit code: $($process.ExitCode))" -ForegroundColor Red
    }
    Write-Host ""
}

Convert-Video -Quality "1080p" -Scale "scale=1920:-2" -VideoBitrate "5000k" -AudioBitrate "192k"
Convert-Video -Quality "720p" -Scale "scale=1280:-2" -VideoBitrate "2500k" -AudioBitrate "128k"
Convert-Video -Quality "480p" -Scale "scale=854:-2" -VideoBitrate "1000k" -AudioBitrate "96k"

Write-Host "========================================" -ForegroundColor Green
Write-Host "  Conversion completada!" -ForegroundColor Green
Write-Host "  Archivos en: $outDir" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Read-Host "Presiona Enter para salir"
