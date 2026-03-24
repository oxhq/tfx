[CmdletBinding()]
param(
    [string]$Version = $env:TFX_VERSION,
    [string]$InstallDir = $(if ($env:TFX_INSTALL_DIR) { $env:TFX_INSTALL_DIR } else { Join-Path $HOME "AppData\Local\Programs\tfx" }),
    [string]$Repo = $(if ($env:TFX_REPO) { $env:TFX_REPO } else { "oxhq/tfx" })
)

$ErrorActionPreference = "Stop"

function Get-LatestVersion {
    param([string]$Repository)
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repository/releases/latest"
    return $release.tag_name
}

function Get-TargetArch {
    switch ($env:PROCESSOR_ARCHITECTURE.ToLowerInvariant()) {
        "amd64" { return "amd64" }
        "arm64" { return "arm64" }
        default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
    }
}

if (-not $Version) {
    $Version = Get-LatestVersion -Repository $Repo
}

$Arch = Get-TargetArch
$Asset = "tfx_{0}_windows_{1}.zip" -f $Version.TrimStart("v"), $Arch
$Url = "https://github.com/$Repo/releases/download/$Version/$Asset"
$TempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("tfx-install-" + [System.Guid]::NewGuid().ToString("N"))
$ZipPath = Join-Path $TempRoot $Asset

New-Item -ItemType Directory -Force -Path $TempRoot | Out-Null
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null

try {
    Write-Host "Downloading $Url"
    Invoke-WebRequest -Uri $Url -OutFile $ZipPath
    Expand-Archive -Path $ZipPath -DestinationPath $TempRoot -Force

    $StageDir = Join-Path $TempRoot ("tfx_{0}_windows_{1}" -f $Version.TrimStart("v"), $Arch)
    Copy-Item (Join-Path $StageDir "tfx.exe") (Join-Path $InstallDir "tfx.exe") -Force
    Write-Host "Installed tfx $Version to $(Join-Path $InstallDir 'tfx.exe')"
}
finally {
    if (Test-Path $TempRoot) {
        Remove-Item -Recurse -Force $TempRoot
    }
}
