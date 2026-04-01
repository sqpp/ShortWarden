param(
  [string]$PostgresDsn = "postgres://shortwarden:shortwarden@localhost:5432/shortwarden?sslmode=disable"
)

$env:SHORTWARDEN_POSTGRES_DSN = $PostgresDsn
go run .\cmd\migrate

