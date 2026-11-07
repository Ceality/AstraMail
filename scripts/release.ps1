$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

$versionLine = Select-String -Path 'internal/version/version.go' -Pattern 'Version = "(.*)"'
$version = $versionLine.Matches.Groups[1].Value
if (-not $version) {
    throw 'Cannot parse version from internal/version/version.go'
}

$commit = 'nogit'
try {
    $resolved = (git rev-parse --short HEAD 2>$null)
    if ($LASTEXITCODE -eq 0 -and $resolved) { $commit = $resolved.Trim() }
} catch {
    $commit = 'nogit'
}
$buildDate = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
$ldflags = "-s -w -X Astramail/internal/version.Commit=$commit -X Astramail/internal/version.BuildDate=$buildDate"

$dist = Join-Path $root 'dist'
New-Item -ItemType Directory -Force -Path $dist | Out-Null

$targets = @(
    @{ GOOS = 'windows'; GOARCH = 'amd64'; Name = "astramail-$version-windows-amd64.exe" },
    @{ GOOS = 'linux';   GOARCH = 'amd64'; Name = "astramail-$version-linux-amd64" },
    @{ GOOS = 'darwin';  GOARCH = 'arm64'; Name = "astramail-$version-darwin-arm64" }
)

$env:CGO_ENABLED = '0'

foreach ($target in $targets) {
    $env:GOOS = $target.GOOS
    $env:GOARCH = $target.GOARCH
    $output = Join-Path $dist $target.Name

    Write-Host "Building $($target.GOOS)/$($target.GOARCH) -> $($target.Name)"
    go build -trimpath "-ldflags=$ldflags" -o $output .
    if ($LASTEXITCODE -ne 0) {
        throw "Build failed: $($target.Name)"
    }
}

$lines = foreach ($target in $targets) {
    $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath (Join-Path $dist $target.Name)).Hash.ToLowerInvariant()
    "$hash  $($target.Name)"
}

$checksums = Join-Path $dist 'checksums.txt'
[IO.File]::WriteAllText($checksums, ($lines -join "`n") + "`n", [Text.UTF8Encoding]::new($false))

Write-Host ''
Write-Host 'Release artifacts:'
Get-ChildItem -LiteralPath $dist | Select-Object Name, Length | Format-Table -AutoSize
Write-Host "Checksums written to $checksums"
