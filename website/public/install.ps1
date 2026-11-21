# installs sablier on windows: irm https://<site>/install.ps1 | iex
#
#   $env:SABLIER_VERSION      a release tag like v0.1.0, default the latest
#   $env:SABLIER_INSTALL_DIR  where to put it, default %LOCALAPPDATA%\Programs\sablier
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue' # the progress bar makes downloads very slow on 5.1

$repo = 'espcaa/sablier'
$file = 'sablier_windows_amd64.exe' # arm64 windows runs it too

$dir = $env:SABLIER_INSTALL_DIR
if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\sablier' }

if ($env:SABLIER_VERSION) {
	$base = "https://github.com/$repo/releases/download/$env:SABLIER_VERSION"
} else {
	$base = "https://github.com/$repo/releases/latest/download"
}
if ($env:SABLIER_RELEASE_URL) { $base = $env:SABLIER_RELEASE_URL } # for testing against a local build

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

$tmp = Join-Path ([IO.Path]::GetTempPath()) ('sablier-' + [Guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
	Write-Host "downloading $file"
	Invoke-WebRequest -UseBasicParsing -Uri "$base/$file" -OutFile (Join-Path $tmp $file)
	Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile (Join-Path $tmp 'checksums.txt')

	$want = $null
	foreach ($line in Get-Content (Join-Path $tmp 'checksums.txt')) {
		$parts = $line -split '\s+'
		if ($parts.Count -eq 2 -and $parts[1] -eq $file) { $want = $parts[0] }
	}
	$got = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp $file)).Hash.ToLower()
	if (-not $want -or $want -ne $got) { throw "checksum mismatch for $file, not installing it" }

	New-Item -ItemType Directory -Force -Path $dir | Out-Null
	$exe = Join-Path $dir 'sablier.exe'
	Move-Item -Force (Join-Path $tmp $file) $exe
} finally {
	Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

Write-Host "installed $(& $exe version) to $exe"

# on the user's PATH for new terminals
if ($env:OS -eq 'Windows_NT') {
	$path = [Environment]::GetEnvironmentVariable('Path', 'User')
	if (-not (($path -split ';') -contains $dir)) {
		[Environment]::SetEnvironmentVariable('Path', ($path.TrimEnd(';') + ';' + $dir).TrimStart(';'), 'User')
		Write-Host "added $dir to your PATH, open a new terminal to use sablier"
	}
}
Write-Host 'next: sablier login'
