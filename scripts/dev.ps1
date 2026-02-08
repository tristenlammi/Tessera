# Tessera Development Scripts (PowerShell)
# Usage: .\scripts\dev.ps1 <command>

param(
    [Parameter(Position=0)]
    [string]$Command = "help"
)

function Show-Help {
    Write-Host @"
Tessera Development Commands
============================

Usage: .\scripts\dev.ps1 <command>

Commands:
  start       Start all containers (dev mode)
  stop        Stop all containers
  restart     Restart all containers
  logs        View logs from all containers
  logs-api    View backend API logs
  logs-web    View frontend logs
  migrate     Run database migrations
  migrate-new Create a new migration file
  shell-db    Open PostgreSQL shell
  shell-api   Open shell in backend container
  clean       Remove all containers and volumes
  help        Show this help message

Examples:
  .\scripts\dev.ps1 start
  .\scripts\dev.ps1 logs-api
  .\scripts\dev.ps1 shell-db
"@
}

function Start-Dev {
    Write-Host "Starting Tessera development environment..." -ForegroundColor Green
    docker-compose up -d
    Write-Host ""
    Write-Host "Services:" -ForegroundColor Cyan
    Write-Host "  Frontend:    http://localhost:3000"
    Write-Host "  Backend API: http://localhost:8080"
    Write-Host "  MinIO:       http://localhost:9001 (admin console)"
    Write-Host "  Traefik:     http://localhost:8081 (dashboard)"
}

function Stop-Dev {
    Write-Host "Stopping Tessera..." -ForegroundColor Yellow
    docker-compose down
}

function Restart-Dev {
    Stop-Dev
    Start-Dev
}

function Show-Logs {
    docker-compose logs -f
}

function Show-ApiLogs {
    docker-compose logs -f backend
}

function Show-WebLogs {
    docker-compose logs -f frontend
}

function Run-Migrate {
    Write-Host "Running database migrations..." -ForegroundColor Green
    docker-compose exec backend go run ./cmd/migrate up
}

function New-Migration {
    param([string]$Name)
    if (-not $Name) {
        $Name = Read-Host "Enter migration name"
    }
    $timestamp = Get-Date -Format "yyyyMMddHHmmss"
    $upFile = "migrations/${timestamp}_${Name}.up.sql"
    $downFile = "migrations/${timestamp}_${Name}.down.sql"
    New-Item -Path $upFile -ItemType File
    New-Item -Path $downFile -ItemType File
    Write-Host "Created migration files:" -ForegroundColor Green
    Write-Host "  $upFile"
    Write-Host "  $downFile"
}

function Open-DbShell {
    docker-compose exec postgres psql -U tessera -d tessera
}

function Open-ApiShell {
    docker-compose exec backend sh
}

function Clean-All {
    Write-Host "Removing all containers and volumes..." -ForegroundColor Red
    docker-compose down -v
    Remove-Item -Recurse -Force .docker-data -ErrorAction SilentlyContinue
    Write-Host "Cleaned!" -ForegroundColor Green
}

switch ($Command) {
    "start"       { Start-Dev }
    "stop"        { Stop-Dev }
    "restart"     { Restart-Dev }
    "logs"        { Show-Logs }
    "logs-api"    { Show-ApiLogs }
    "logs-web"    { Show-WebLogs }
    "migrate"     { Run-Migrate }
    "migrate-new" { New-Migration }
    "shell-db"    { Open-DbShell }
    "shell-api"   { Open-ApiShell }
    "clean"       { Clean-All }
    default       { Show-Help }
}
