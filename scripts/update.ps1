$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($env:SHORTWARDEN_COMPOSE_PROJECT_NAME)) {
  $project = "shortwarden"
} else {
  $project = $env:SHORTWARDEN_COMPOSE_PROJECT_NAME
}

$cf = $env:SHORTWARDEN_COMPOSE_FILE
if ([string]::IsNullOrWhiteSpace($cf)) {
  if (Test-Path "docker-compose.nginx.yml") { $cf = "docker-compose.nginx.yml" }
  else { $cf = "docker-compose.yml" }
}

$composeArgs = @("-f", $cf, "-p", $project)

docker compose @composeArgs pull api

$hasScreenshot = $false
try {
  $svc = docker compose @composeArgs config --services 2>$null
  if ($svc) {
    foreach ($line in $svc) {
      if ($line.Trim() -eq "screenshot") { $hasScreenshot = $true; break }
    }
  }
} catch {
  $hasScreenshot = $false
}

if ($hasScreenshot) {
  docker compose @composeArgs build screenshot
  docker compose @composeArgs up -d --no-deps --force-recreate api screenshot
} else {
  docker compose @composeArgs up -d --no-deps --force-recreate api
}

docker compose @composeArgs restart nginx
docker image prune -f

Write-Output "Update completed."
