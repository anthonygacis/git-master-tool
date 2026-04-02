$ErrorActionPreference = "Stop"

$Repo    = "anthonygacis/git-master-tool"
$Binary  = "gitmt"
$ApiUrl  = "https://api.github.com/repos/$Repo/releases/latest"

function Write-Info  { Write-Host "=> $args" -ForegroundColor Cyan }
function Write-Ok    { Write-Host "v $args"  -ForegroundColor Green }
function Write-Warn  { Write-Host "! $args"  -ForegroundColor Yellow }
function Write-Fatal { Write-Host "error: $args" -ForegroundColor Red; exit 1 }

Write-Host "`ngitmt installer`n" -ForegroundColor Cyan

$Arch = if ([System.Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
Write-Info "Platform: windows/$Arch"

Write-Info "Fetching latest release info..."
try {
    $Release = Invoke-RestMethod -Uri $ApiUrl -Headers @{ "User-Agent" = "gitmt-installer" }
} catch {
    Write-Fatal "Could not reach GitHub API: $_"
}
$Version = $Release.tag_name
Write-Ok "Latest release: $Version"

$AssetName   = "$Binary-windows-$Arch.exe"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$AssetName"

$InstallDir = "$env:LOCALAPPDATA\Programs\gitmt"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$Dest = Join-Path $InstallDir "$Binary.exe"

Write-Info "Downloading $AssetName ..."
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $Dest -UseBasicParsing
} catch {
    Write-Fatal "Download failed. Verify the asset '$AssetName' exists at:`n  https://github.com/$Repo/releases/tag/$Version"
}
Write-Ok "Downloaded $Version"

Write-Info "Installing to $Dest ..."

$UserPath = [System.Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [System.Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
    $env:PATH = "$env:PATH;$InstallDir"
    Write-Ok "Added $InstallDir to your user PATH"
} else {
    Write-Ok "$InstallDir already in PATH"
}

if (Get-Command $Binary -ErrorAction SilentlyContinue) {
    Write-Ok "$Binary is ready"
} else {
    Write-Warn "$Binary installed but PATH update requires a new terminal session."
}

Write-Host "`nDone! Run:  gitmt compare develop master`n" -ForegroundColor Green
