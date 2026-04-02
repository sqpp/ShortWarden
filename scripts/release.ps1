param(
  [Parameter(Mandatory = $true)]
  [string]$Version,
  [string]$Image = "sqpp/shortwarden"
)

$ErrorActionPreference = "Stop"

$gitSha = "unknown"
try {
  $gitSha = (git rev-parse --short HEAD).Trim()
} catch {
  $gitSha = "unknown"
}

$buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

Write-Output "Building $Image`:$Version"
docker build -f Dockerfile.api `
  --build-arg APP_VERSION=$Version `
  --build-arg GIT_SHA=$gitSha `
  --build-arg BUILD_TIME=$buildTime `
  -t "$Image`:$Version" `
  -t "$Image`:latest" `
  .

Write-Output "Pushing $Image`:$Version"
docker push "$Image`:$Version"
Write-Output "Pushing $Image`:latest"
docker push "$Image`:latest"

Write-Output "Release completed: $Image`:$Version"
