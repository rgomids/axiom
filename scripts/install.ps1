#requires -Version 5.1
# Invoke-Expression otherwise binds parameters in the caller's scope, including
# validating an omitted Channel as an empty string. Always use a child scope.
& {
[CmdletBinding()]
param(
    [string]$Version,
    [ValidateSet('stable')][string]$Channel,
    [string]$BinDir = (Join-Path $env:USERPROFILE '.axiom\windows\bin'),
    [string]$ReceiptDir = (Join-Path $env:USERPROFILE '.axiom\windows\install'),
    [switch]$SkipRuntimeSetup,
    [switch]$SessionOnly
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
# Keep an existing default installation at its recorded location. A fresh
# installation avoids AppData ancestors without rewriting their permissions.
$legacyBin = Join-Path $env:LOCALAPPDATA 'Axiom\bin'
$legacyReceipt = Join-Path $env:LOCALAPPDATA 'Axiom\install'
if (-not ((Test-Path -LiteralPath (Join-Path $env:USERPROFILE '.axiom\windows\bin\axiom.exe')) -or
          (Test-Path -LiteralPath (Join-Path $env:USERPROFILE '.axiom\windows\install\installation.receipt'))) -and
    ((Test-Path -LiteralPath (Join-Path $legacyBin 'axiom.exe')) -or
     (Test-Path -LiteralPath (Join-Path $legacyReceipt 'installation.receipt')))) {
    if (-not $PSBoundParameters.ContainsKey('BinDir')) { $BinDir = $legacyBin }
    if (-not $PSBoundParameters.ContainsKey('ReceiptDir')) { $ReceiptDir = $legacyReceipt }
}
$tagPattern = '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.(0|[1-9][0-9]*))?$'
if ($Version -and $Channel) { throw 'Choose -Version or -Channel, not both.' }
if ($Version -and $Version -cnotmatch $tagPattern) { throw 'An exact vMAJOR.MINOR.PATCH[-rc.N] version is required.' }
$architecture = $env:PROCESSOR_ARCHITECTURE
if ($env:PROCESSOR_ARCHITEW6432) { $architecture = $env:PROCESSOR_ARCHITEW6432 }
if ($env:OS -ne 'Windows_NT' -or $architecture -ne 'AMD64' -or -not [Environment]::Is64BitProcess) {
    throw 'Windows amd64 and 64-bit PowerShell are required.'
}
# Eligibility is workstation or member Server, never the numeric OS version.
$productType = (Get-CimInstance Win32_OperatingSystem).ProductType
if ($productType -ne 1 -and $productType -ne 3) {
    throw 'Windows workstation or member Server is required; domain controllers and unknown product types are unsupported.'
}
if (-not (Get-Command tar.exe -ErrorAction SilentlyContinue)) {
    throw 'The Windows tar.exe utility is required.'
}
foreach ($path in @($BinDir, $ReceiptDir)) {
    if ($path -notmatch '^[A-Za-z]:\\' -or $path -match '[\r\n=;]') {
        throw 'Installation directories must be absolute local drive paths.'
    }
}

function Update-AxiomUserPath([string]$Directory) {
    # Read raw registry text: preserve expandable entries and the existing kind.
    # Only this user's PATH is touched, never the machine environment.
    $userEnvironment = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
    try {
        $options = [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames
        $original = $userEnvironment.GetValue('Path',$null,$options)
        $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
        if ($null -ne $original) {
            $kind = $userEnvironment.GetValueKind('Path')
            if ($original -isnot [string] -or $kind -notin @([Microsoft.Win32.RegistryValueKind]::String,[Microsoft.Win32.RegistryValueKind]::ExpandString)) {
                throw 'User PATH has an unsupported registry value; it was preserved.'
            }
        }
        foreach ($entry in @($original -split ';')) {
            $expanded = [Environment]::ExpandEnvironmentVariables($entry.Trim().Trim('"')).TrimEnd('\')
            if ($expanded -ieq $Directory.TrimEnd('\')) { return }
        }
        $updated = if ([string]::IsNullOrEmpty($original)) { $Directory } elseif ($original.EndsWith(';')) { $original + $Directory } else { $original + ';' + $Directory }
        if ($updated.Length -ge 32767) { throw 'User PATH would exceed the Windows environment limit; it was preserved.' }
        if ($userEnvironment.GetValue('Path',$null,$options) -cne $original) { throw 'User PATH changed during setup; retry without overwriting the new value.' }
        $userEnvironment.SetValue('Path',$updated,$kind)
        if ($userEnvironment.GetValue('Path',$null,$options) -cne $updated) { throw 'User PATH verification failed; inspect it before retrying.' }
    } finally { $userEnvironment.Dispose() }
    # Notify Explorer and other listeners so newly launched applications can
    # pick up the user environment. Existing terminals keep their own snapshot.
    if (-not ('AxiomEnvironmentNotification' -as [type])) {
        Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class AxiomEnvironmentNotification {
    [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
    private static extern IntPtr SendMessageTimeout(IntPtr window, uint message, IntPtr wparam, string lparam, uint flags, uint timeout, out UIntPtr result);
    public static void Notify() {
        UIntPtr result;
        SendMessageTimeout(new IntPtr(0xffff), 0x001a, IntPtr.Zero, "Environment", 2, 1000, out result);
    }
}
'@
    }
    [AxiomEnvironmentNotification]::Notify()
}

# Follow redirects explicitly so an HTTPS response can never downgrade to HTTP.
# Bound every response before it reaches disk; the client timeout bounds each hop.
Add-Type -AssemblyName System.Net.Http
$handler = New-Object System.Net.Http.HttpClientHandler
$handler.AllowAutoRedirect = $false
$client = New-Object System.Net.Http.HttpClient($handler)
$client.Timeout = [TimeSpan]::FromSeconds(120)
$client.DefaultRequestHeaders.UserAgent.ParseAdd('Axiom-Windows-Installer')
function Get-ReleaseFile([Uri]$Uri, [string]$Target, [long]$Limit) {
    $cancellation = New-Object Threading.CancellationTokenSource
    $cancellation.CancelAfter(300000)
    try {
    for ($hop = 0; $hop -lt 10; $hop++) {
        if ($Uri.Scheme -cne 'https' -or $Uri.UserInfo) { throw 'Only HTTPS release URLs without credentials are permitted.' }
        $response = $client.GetAsync($Uri, [System.Net.Http.HttpCompletionOption]::ResponseHeadersRead, $cancellation.Token).GetAwaiter().GetResult()
        try {
            if ([int]$response.StatusCode -in @(301,302,303,307,308)) {
                $location = $response.Headers.Location
                if (-not $location) { throw 'Release redirect has no location.' }
                $Uri = New-Object Uri($Uri, $location)
                continue
            }
            $response.EnsureSuccessStatusCode() | Out-Null
            if ($response.Content.Headers.ContentLength -gt $Limit) { throw 'Release response exceeds its size limit.' }
            $source = $response.Content.ReadAsStreamAsync().GetAwaiter().GetResult()
            $destination = [IO.File]::Open($Target, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
            try {
                $buffer = New-Object byte[] 65536
                [long]$total = 0
                $deadline = [DateTime]::UtcNow.AddSeconds(300)
                while (($read = $source.ReadAsync($buffer,0,$buffer.Length,$cancellation.Token).GetAwaiter().GetResult()) -gt 0) {
                    $total += $read
                    if ($total -gt $Limit -or [DateTime]::UtcNow -gt $deadline) { throw 'Release response exceeds download limits.' }
                    $destination.Write($buffer,0,$read)
                }
            } finally { $destination.Dispose(); $source.Dispose() }
            return
        } finally { $response.Dispose() }
    }
    throw 'Too many release redirects.'
    } finally { $cancellation.Dispose() }
}

function Assert-SafeStagingParent([string]$Path) {
    if ($Path -notmatch '^[A-Za-z]:\\' -or $Path -match '[\r\n]') { throw 'A local user profile is required for staging.' }
    $drive = New-Object IO.DriveInfo([IO.Path]::GetPathRoot($Path))
    if ($drive.DriveType -ne 'Fixed' -or $drive.DriveFormat -ne 'NTFS') { throw 'Staging requires local NTFS storage.' }
    $current = Get-Item -LiteralPath $Path -Force
    $owner = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    $trusted = @($owner, 'S-1-5-18', 'S-1-5-32-544', 'S-1-5-80-956008885-3418522649-1831038044-1853292631-2271478464')
    while ($null -ne $current) {
        if ($current.Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Reparse points are not supported in staging paths.' }
        $security = Get-Acl -LiteralPath $current.FullName
        if ($trusted -notcontains $security.GetOwner([Security.Principal.SecurityIdentifier]).Value) { throw 'Staging ancestor has an untrusted owner.' }
        # DELETE, WRITE_DAC, WRITE_OWNER and DELETE_CHILD permit replacement.
        foreach ($ace in $security.GetAccessRules($true,$true,[Security.Principal.SecurityIdentifier])) {
            if ($ace.PropagationFlags -band [Security.AccessControl.PropagationFlags]::InheritOnly) { continue }
            if ($ace.AccessControlType -eq 'Allow' -and $trusted -notcontains $ace.IdentityReference.Value -and (([long]$ace.FileSystemRights -band 0x000D0040) -ne 0)) {
                throw 'Staging ancestor permits replacement by another principal.'
            }
        }
        $current = $current.Parent
    }
}

# Protect staging before storing executable bytes. No global ACL or PATH changes.
Assert-SafeStagingParent $env:USERPROFILE
$work = Join-Path $env:USERPROFILE ('axiom-install-' + [guid]::NewGuid().ToString('N'))
try {
    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetOwner($sid)
    $acl.SetAccessRuleProtection($true,$false)
    $rule = New-Object Security.AccessControl.FileSystemAccessRule($sid,'FullControl','ContainerInherit,ObjectInherit','None','Allow')
    $acl.AddAccessRule($rule)
    # .NET Framework's security overload supplies the DACL at creation.
    # PowerShell 7 uses the equivalent FileSystemAclExtensions API.
    if ($PSVersionTable.PSEdition -eq 'Core') {
        [IO.FileSystemAclExtensions]::CreateDirectory($acl, $work) | Out-Null
    } else {
        [IO.Directory]::CreateDirectory($work, $acl) | Out-Null
    }

    $repository = 'https://github.com/rgomids/axiom'
    if ($Version) { $tag = $Version } else {
        $metadata = Join-Path $work 'latest.json'
        Get-ReleaseFile 'https://api.github.com/repos/rgomids/axiom/releases/latest' $metadata 1048576
        $release = Get-Content -LiteralPath $metadata -Raw | ConvertFrom-Json
        if ($release -isnot [pscustomobject] -or $release.tag_name -isnot [string] -or $release.draft -isnot [bool] -or $release.prerelease -isnot [bool]) {
            throw 'Invalid release metadata.'
        }
        if ($release.draft -or $release.prerelease) { throw 'No stable release was resolved.' }
        $tag = [string]$release.tag_name
    }
    if ($tag -cnotmatch $tagPattern -or (-not $Version -and $tag.Contains('-'))) { throw 'Invalid release tag.' }
    $bundle = "axiom-$($tag.Substring(1))-windows-amd64"
    $asset = "$bundle.tar.gz"
    $sums = Join-Path $work 'SHA256SUMS'
    $archive = Join-Path $work $asset
    Get-ReleaseFile "$repository/releases/download/$tag/SHA256SUMS" $sums 65536
    $entries = @(Get-Content -LiteralPath $sums | Where-Object {
        $fields = $_ -split '\s+'
        $fields.Count -ge 2 -and $fields[1] -ceq $asset
    })
    if ($entries.Count -ne 1 -or $entries[0] -cnotmatch ("^[a-f0-9]{64}  " + [regex]::Escape($asset) + '$')) {
        throw 'Release checksum is unavailable, malformed or ambiguous.'
    }
    Get-ReleaseFile "$repository/releases/download/$tag/$asset" $archive 268435456
    if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -cne $entries[0].Substring(0,64)) {
        throw 'Release checksum mismatch.'
    }
    # Extract only the exact executable entry into private staging. Refuse every
    # link/special entry and ambiguous executable before invoking tar extraction.
    $names = @(& tar.exe -tzf $archive)
    if ($LASTEXITCODE -ne 0) { throw 'Invalid release archive.' }
    $types = @(& tar.exe -tvzf $archive)
    if ($LASTEXITCODE -ne 0 -or $names.Count -gt 64 -or @($types | Where-Object { $_ -notmatch '^[-d]' }).Count -ne 0) {
        throw 'Archive links and special entries are refused.'
    }
    if (@($names | Where-Object { $_ -ceq "$bundle/axiom.exe" }).Count -ne 1) {
        throw 'Archive executable is missing or ambiguous.'
    }
    foreach ($name in $names) {
        if (-not $name.StartsWith("$bundle/",[StringComparison]::Ordinal) -and $name -cne $bundle) { throw 'Invalid archive root.' }
        if ($name -match '\\|(^|/)\.\.?(/|$)|:|[\r\n]') { throw 'Unsafe archive path.' }
    }
    & tar.exe -xzf $archive -C $work -- "$bundle/axiom.exe"
    if ($LASTEXITCODE -ne 0) { throw 'Could not extract the verified executable.' }
    & (Join-Path $work "$bundle\axiom.exe") install-release --archive $archive --checksums $sums --bin-dir $BinDir --receipt-dir $ReceiptDir
    if ($LASTEXITCODE -ne 0) { throw "Verified installer failed (exit $LASTEXITCODE)." }
    $installed = Join-Path $BinDir 'axiom.exe'
    & $installed version
    if ($LASTEXITCODE -ne 0) { throw 'Installed executable verification failed.' }
    if (-not $SkipRuntimeSetup) {
        & $installed first-run
        if ($LASTEXITCODE -ne 0) { throw 'Binary installed; Runtime setup failed. Resolve the reported conflict and run axiom first-run again.' }
    }
    if (-not $SessionOnly) {
        try { Update-AxiomUserPath $BinDir }
        catch { throw "Binary installed; user PATH setup failed: $($_.Exception.Message)" }
        Write-Output "user_path=verified; directory=$BinDir"
    }
    # Make it usable immediately as well as in future user environments.
    if (@($env:PATH -split ';' | Where-Object { $_.TrimEnd('\') -ieq $BinDir.TrimEnd('\') }).Count -eq 0) {
        $env:PATH = "$BinDir;$env:PATH"
    }
    if ($SkipRuntimeSetup) { Write-Output "onboarding_status=binary_only; binary=$installed" }
    else { Write-Output "onboarding_status=ready; binary=$installed" }
} finally {
    $client.Dispose()
    $handler.Dispose()
    # This exact UUID-named staging directory was created by this invocation.
    if (Test-Path -LiteralPath $work) { Remove-Item -LiteralPath $work -Recurse -Force }
}
} @args
