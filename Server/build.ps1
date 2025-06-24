# Create build directory if it doesn't exist
if (-not (Test-Path -Path "./build")) {
    New-Item -ItemType Directory -Path "./build" | Out-Null
}

Write-Host "Building Windows executable..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -o ./build/main.exe

Write-Host "Building Linux executable..."
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -o ./build/simplehost

Write-Host "Builds complete! Files are in the build/ directory."