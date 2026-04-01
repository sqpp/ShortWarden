param(
  [string]$HttpAddr = ":8080",
  [string]$PublicBaseUrl = "http://localhost:8080",
  [string]$PostgresDsn = "postgres://shortwarden:shortwarden@localhost:5432/shortwarden?sslmode=disable",
  [string]$JwtSecret = "dev-secret-change-me"
)

$env:SHORTWARDEN_HTTP_ADDR = $HttpAddr
$env:SHORTWARDEN_PUBLIC_BASE_URL = $PublicBaseUrl
$env:SHORTWARDEN_POSTGRES_DSN = $PostgresDsn
$env:SHORTWARDEN_JWT_SECRET = $JwtSecret
$env:SHORTWARDEN_COOKIE_SECURE = "0"

go run .\cmd\shortwarden

