# Downloads the official sing-box for Windows (sing-box.exe + libcronet.dll) into this folder.
# Used automatically on first run when sing-box.exe is missing. Safe to run by hand any time.
param(
    [switch]$Quiet,
    [switch]$Force,
    [string]$BaseUrl = 'https://github.com/SagerNet/sing-box/releases/download'
)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'   # progress bars make downloads very slow on Windows PowerShell 5.1
$Dir     = $PSScriptRoot
$Ver     = '1.14.0'
$LogFile = Join-Path $Dir 'get-singbox.log'
# SHA-256 of the official release zips for $Ver
$Hashes  = @{
    'amd64' = '3ffb56267da14e287be48bd10cf7e6505260125bad940b75101fbb4d5d58e5d6'
    'arm64' = 'f58dff882b2feb022da8de41943804b38681ecab5e1f490f23602fc37e9d5dd4'
}

function Log([string]$m) {
    Add-Content -Path $LogFile -Value ("{0} {1}" -f (Get-Date -Format 'yyyy-MM-dd HH:mm:ss'), $m) -ErrorAction SilentlyContinue
    if (-not $Quiet) { Write-Host $m }
}

try { [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12 } catch {}

$exePath = Join-Path $Dir 'sing-box.exe'
if ((Test-Path $exePath) -and -not $Force) { Log 'sing-box.exe is already here - nothing to do.'; exit 0 }

$archRaw = $env:PROCESSOR_ARCHITEW6432
if (-not $archRaw) { $archRaw = $env:PROCESSOR_ARCHITECTURE }
switch -Regex ("$archRaw") {
    'ARM64'     { $Plat = 'arm64' }
    '^x86$|^386$' { $Plat = '386' }
    default     { $Plat = 'amd64' }
}

$Tmp = Join-Path ([IO.Path]::GetTempPath()) ('hr_sb_' + [guid]::NewGuid().ToString('N'))
try {
    New-Item -ItemType Directory -Path $Tmp -Force | Out-Null
    $zip      = Join-Path $Tmp 'sb.zip'
    $expected = $Hashes[$Plat]
    $url      = "$BaseUrl/v$Ver/sing-box-$Ver-windows-$Plat.zip"

    Log "Downloading sing-box $Ver for Windows ($Plat) - about 35 MB, please wait..."
    try {
        Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing -TimeoutSec 600
    } catch {
        Log "That version didn't download ($($_.Exception.Message)) - trying the latest release instead..."
        $rel   = Invoke-RestMethod -Uri 'https://api.github.com/repos/SagerNet/sing-box/releases/latest' -UseBasicParsing -Headers @{ 'User-Agent' = 'Home-Reality' } -TimeoutSec 60
        $asset = $rel.assets | Where-Object { $_.name -like "*-windows-$Plat.zip" } | Select-Object -First 1
        if (-not $asset) { throw "No windows-$Plat build found in the latest release." }
        $expected = $null
        if ($asset.digest -and "$($asset.digest)" -like 'sha256:*') { $expected = "$($asset.digest)".Substring(7) }
        Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $zip -UseBasicParsing -TimeoutSec 600
    }

    if ($expected) {
        $got = (Get-FileHash -Path $zip -Algorithm SHA256).Hash
        if ($got -ine $expected) { throw "Checksum mismatch - the download was corrupted or tampered with. Not installing it." }
        Log 'Checksum OK.'
    } else {
        Log 'No checksum available for this build - continuing (download was over HTTPS from github.com).'
    }

    $ext = Join-Path $Tmp 'x'
    Expand-Archive -Path $zip -DestinationPath $ext -Force
    $exe = Get-ChildItem -Path $ext -Recurse -Filter 'sing-box.exe' | Select-Object -First 1
    if (-not $exe) { throw 'sing-box.exe was not found inside the download.' }
    $dll = Get-ChildItem -Path $ext -Recurse -Filter 'libcronet.dll' | Select-Object -First 1

    # copy under a temporary name first so nothing ever sees a half-written sing-box.exe
    if ($dll) { Copy-Item $dll.FullName (Join-Path $Dir 'libcronet.dll') -Force }
    $part = Join-Path $Dir 'sing-box.exe.part'
    Copy-Item $exe.FullName $part -Force
    Move-Item $part $exePath -Force
    try { Unblock-File $exePath; if ($dll) { Unblock-File (Join-Path $Dir 'libcronet.dll') } } catch {}

    if ($IsWindows -ne $false) {
        $v = (& $exePath version 2>&1 | Select-Object -First 1)
        Log "Installed: $v"
    } else {
        Log 'Installed (version check skipped - not Windows).'
    }
    exit 0
} catch {
    Log "ERROR: $($_.Exception.Message)"
    exit 1
} finally {
    Remove-Item -Path $Tmp -Recurse -Force -ErrorAction SilentlyContinue
}
