# install.ps1
$version = Invoke-RestMethod -Uri "https://api.github.com/repos/aooohan/version-fox/releases/latest" | Select-Object -ExpandProperty tag_name
$osType = "windows"

# 检测操作系统架构
if ([IntPtr]::Size -eq 8) {
  $archType = "amd64"
} else {
  $archType = "386"
}

$fileName = "version-fox_${version}_${osType}_${archType}.zip"
$url = "https://github.com/aooohan/version-fox/releases/download/$version/$fileName"

Write-Host "Downloading vfox $version ..."
try {
    Invoke-WebRequest -Uri $url -OutFile $fileName
} catch {
    Write-Host "Failed to download vfox. Please check your network connection and try again."
    exit 1
}
Write-Host "Extracting vfox ..."
Expand-Archive -Path $fileName -DestinationPath .

$destDir = "C:\Program Files\version-fox"
if (!(Test-Path -Path $destDir)) {
    New-Item -ItemType Directory -Path $destDir | Out-Null
}

Write-Host "Moving vfox to $destDir ..."
Move-Item -Path .\vfox -Destination $destDir

$envPath = [Environment]::GetEnvironmentVariable("Path", "Machine")
if (!$envPath.Contains($destDir)) {
    [Environment]::SetEnvironmentVariable("Path", $envPath + ";$destDir", "Machine")
}

Remove-Item $fileName
Write-Host "vfox installed successfully!"