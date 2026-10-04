#!/usr/bin/env pwsh
#
#    Copyright 2026 Han Li and contributors
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.

<#
.SYNOPSIS
    Generates the MSIX tile icons (msix/assets) from the repository logo.

.DESCRIPTION
    The AppxManifest requires raster PNG logos (SVG is rejected by AppX
    deployment), but those binaries are not committed: msix/make-msix.ps1 runs
    this script right before packing, so the icons are always derived from the
    canonical logo at build time. Transparent margins are cropped away and the
    artwork is scaled (high-quality bicubic) onto a transparent square canvas.

    Requires Windows (System.Drawing).

.PARAMETER LogoPath
    Path to the source logo PNG.

.PARAMETER OutDir
    Output directory for the generated icons.
#>

param(
    [Parameter(Mandatory = $true)]
    [string]$LogoPath,

    [Parameter(Mandatory = $true)]
    [string]$OutDir
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Drawing

$targets = @(
    @{ Name = "Square44x44Logo.png"; Size = 44 },
    @{ Name = "Square150x150Logo.png"; Size = 150 },
    @{ Name = "StoreLogo.png"; Size = 50 }
)

$src = [System.Drawing.Bitmap]::new((Resolve-Path $LogoPath).Path)
try {
    # Locate the artwork bounding box via the alpha channel (LockBits: fast).
    $bounds = [System.Drawing.Rectangle]::new(0, 0, $src.Width, $src.Height)
    $data = $src.LockBits($bounds, [System.Drawing.Imaging.ImageLockMode]::ReadOnly, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    try {
        $minX, $minY, $maxX, $maxY = $src.Width, $src.Height, -1, -1
        $bytes = New-Object byte[] ($data.Stride * $src.Height)
        [System.Runtime.InteropServices.Marshal]::Copy($data.Scan0, $bytes, 0, $bytes.Length)
        for ($y = 0; $y -lt $src.Height; $y++) {
            $rowBase = $y * $data.Stride
            for ($x = 0; $x -lt $src.Width; $x++) {
                if ($bytes[$rowBase + $x * 4 + 3] -gt 8) { # BGRA: alpha at +3
                    if ($x -lt $minX) { $minX = $x }
                    if ($y -lt $minY) { $minY = $y }
                    if ($x -gt $maxX) { $maxX = $x }
                    if ($y -gt $maxY) { $maxY = $y }
                }
            }
        }
    }
    finally {
        $src.UnlockBits($data)
    }

    if ($maxX -lt 0) {
        $crop = $bounds
    }
    else {
        $crop = [System.Drawing.Rectangle]::new($minX, $minY, $maxX - $minX + 1, $maxY - $minY + 1)
    }
    Write-Host "source $LogoPath`: canvas $($src.Width)x$($src.Height), content $($crop.Width)x$($crop.Height)"

    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

    foreach ($target in $targets) {
        $size = $target.Size
        $dst = [System.Drawing.Bitmap]::new($size, $size)
        try {
            $graphics = [System.Drawing.Graphics]::FromImage($dst)
            try {
                $graphics.Clear([System.Drawing.Color]::Transparent)
                $graphics.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $graphics.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
                $graphics.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality

                $scale = [Math]::Min($size / $crop.Width, $size / $crop.Height)
                $width = [Math]::Max(1, [int][Math]::Round($crop.Width * $scale))
                $height = [Math]::Max(1, [int][Math]::Round($crop.Height * $scale))
                $offsetX = [int](($size - $width) / 2)
                $offsetY = [int](($size - $height) / 2)

                # TileFlipXY prevents translucent ghosting from bicubic edge sampling.
                $attributes = [System.Drawing.Imaging.ImageAttributes]::new()
                $attributes.SetWrapMode([System.Drawing.Drawing2D.WrapMode]::TileFlipXY)

                $destRect = [System.Drawing.Rectangle]::new($offsetX, $offsetY, $width, $height)
                $graphics.DrawImage($src, $destRect, $crop.X, $crop.Y, $crop.Width, $crop.Height, [System.Drawing.GraphicsUnit]::Pixel, $attributes)
            }
            finally {
                $graphics.Dispose()
            }
            $outPath = Join-Path $OutDir $target.Name
            $dst.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)
            Write-Host "wrote $outPath ($size x $size)"
        }
        finally {
            $dst.Dispose()
        }
    }
}
finally {
    $src.Dispose()
}
