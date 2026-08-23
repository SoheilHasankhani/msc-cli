# Download the latest msc GitHub Release for this Windows arch, verify
# checksums.txt, and install msc.exe. Override with MSC_REPO, MSC_VERSION,
# MSC_INSTALL_DIR, MSC_GITHUB_TOKEN.
$ErrorActionPreference = "Stop"

$Repo = if ($env:MSC_REPO) { $env:MSC_REPO } else { "SoheilHasankhani/msc-cli" }
$InstallDir = if ($env:MSC_INSTALL_DIR) { $env:MSC_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "msc" }

$arch = $env:PROCESSOR_ARCHITECTURE
switch -Regex ($arch) {
    "AMD64|X64" { $goarch = "amd64" }
    "ARM64" { $goarch = "arm64" }
    default { throw "msc install: unsupported architecture: $arch" }
}

$apiHeaders = @{
    Accept       = "application/vnd.github+json"
    "User-Agent" = "msc-cli-install"
}
$dlHeaders = @{
    "User-Agent" = "msc-cli-install"
}
if ($env:MSC_GITHUB_TOKEN) {
    $apiHeaders["Authorization"] = "Bearer $($env:MSC_GITHUB_TOKEN)"
    $dlHeaders["Authorization"] = "Bearer $($env:MSC_GITHUB_TOKEN)"
}

if ($env:MSC_VERSION) {
    $tag = $env:MSC_VERSION
    if (-not $tag.StartsWith("v")) { $tag = "v$tag" }
} else {
    $latest = Invoke-RestMethod -Headers $apiHeaders -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $tag = $latest.tag_name
    if (-not $tag) { throw "msc install: could not read latest tag for $Repo" }
}

$version = $tag.TrimStart("v")
$asset = "msc_${version}_windows_${goarch}.zip"
$base = "https://github.com/$Repo/releases/download/$tag"

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("msc-install-" + [guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "msc install: downloading $asset from $Repo $tag"
    $zipPath = Join-Path $tmp $asset
    $sumPath = Join-Path $tmp "checksums.txt"
    Invoke-WebRequest -Headers $dlHeaders -Uri "$base/$asset" -OutFile $zipPath
    Invoke-WebRequest -Headers $dlHeaders -Uri "$base/checksums.txt" -OutFile $sumPath

    $want = $null
    Get-Content $sumPath | ForEach-Object {
        $parts = $_ -split "\s+"
        if ($parts.Count -ge 2 -and $parts[-1] -eq $asset) { $want = $parts[0].ToLowerInvariant() }
    }
    if (-not $want) { throw "msc install: checksums.txt has no entry for $asset" }

    $sha = [System.Security.Cryptography.SHA256]::Create()
    $stream = [System.IO.File]::OpenRead($zipPath)
    try {
        $got = [BitConverter]::ToString($sha.ComputeHash($stream)).Replace("-", "").ToLowerInvariant()
    } finally {
        $stream.Dispose()
        $sha.Dispose()
    }
    if ($got -ne $want) {
        throw "msc install: checksum mismatch for $asset`n  want $want`n  got  $got"
    }

    $out = Join-Path $tmp "out"
    Expand-Archive -Path $zipPath -DestinationPath $out -Force
    $bin = Get-ChildItem -Path $out -Recurse -Filter "msc.exe" | Select-Object -First 1
    if (-not $bin) { throw "msc install: archive did not contain msc.exe" }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $dest = Join-Path $InstallDir "msc.exe"
    Copy-Item -Path $bin.FullName -Destination $dest -Force
    Write-Host "installed $dest ($tag)"

    $env:MSC_INSTALL_DIR = $InstallDir
    try {
        & $dest path install
    } catch {
        Write-Host "msc install: could not configure PATH automatically; add $InstallDir to PATH"
    }
    & $dest completion install
    & $dest --version
    Write-Host "register a project with: msc init --repo <git-ssh-url> --path <meta-repo>"
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
