$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$version = "v0.2.0" 
$exeName = "zaptun-windows-$arch.exe"
$installDir = "$env:ProgramFiles\Zaptun"
$exePath = "$installDir\zaptun.exe"
$downloadUrl = "https://github.com/harsh082ip/ZapTun/releases/download/$version/$exeName"

Write-Host "Downloading $exeName from $downloadUrl..."

if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Invoke-WebRequest -Uri $downloadUrl -OutFile $exePath

$envPath = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
if ($envPath -notlike "*$installDir*") {
    setx /M PATH "$envPath;$installDir"
    Write-Host "Added $installDir to system PATH. Please restart your terminal."
}

Write-Host "`n✅ Zaptun installed successfully!"
Write-Host "You can now run 'zaptun' from any terminal."
