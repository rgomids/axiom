#requires -Version 5.1
# Execute the production bootstrap with an in-memory HTTP transport. Host data
# is injected only in this test; production exposes no support-policy override.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http
Add-Type -ReferencedAssemblies System.Net.Http -TypeDefinition @'
using System;
using System.Collections.Generic;
using System.Net;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
public class BootstrapTransport : HttpMessageHandler {
    public Dictionary<string, byte[]> Files = new Dictionary<string, byte[]>();
    public List<string> Requests = new List<string>();
    public string Redirect;
    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken token) {
        string url = request.RequestUri.AbsoluteUri;
        Requests.Add(url);
        var response = new HttpResponseMessage(HttpStatusCode.OK);
        if (Redirect != null) {
            response.StatusCode = HttpStatusCode.Redirect;
            response.Headers.Location = new Uri(Redirect);
        } else {
            if (!Files.ContainsKey(url)) throw new Exception("Unexpected request: " + url);
            response.Content = new ByteArrayContent(Files[url]);
        }
        return Task.FromResult(response);
    }
}
'@
$source = [IO.File]::ReadAllText((Join-Path $PSScriptRoot 'install.ps1'))
$construction = '$client = New-Object System.Net.Http.HttpClient($handler)'
if (-not $source.Contains($construction)) { throw 'Bootstrap transport seam changed.' }
$bootstrap = [scriptblock]::Create($source.Replace($construction, '$client = $testClient'))
$testRoot = $env:USERPROFILE
$work = Join-Path $testRoot ('axiom-bootstrap-test-' + [guid]::NewGuid().ToString('N'))
[IO.Directory]::CreateDirectory($work) | Out-Null
$bin = Join-Path $work 'bin'
$receipt = Join-Path $work 'receipt'
$asset = 'axiom-1.0.0-windows-amd64.tar.gz'
$base = 'https://github.com/rgomids/axiom/releases/download/v1.0.0/'
$utf8 = New-Object Text.UTF8Encoding($false)
$savedArch = $env:PROCESSOR_ARCHITECTURE
$savedWow = $env:PROCESSOR_ARCHITEW6432
$savedProfile = $env:USERPROFILE
$script:productType = 1
function Get-CimInstance { param($ClassName) [pscustomobject]@{ProductType=$script:productType;Version='10.0.22621'} }

# Build tiny tar.gz fixtures without invoking any executable from the archive.
function New-ArchiveBytes([string[]]$Names) {
    $raw = New-Object IO.MemoryStream
    foreach ($name in $Names) {
        $header = New-Object byte[] 512
        [Text.Encoding]::ASCII.GetBytes($name).CopyTo($header,0)
        [Text.Encoding]::ASCII.GetBytes("0000700`0").CopyTo($header,100)
        [Text.Encoding]::ASCII.GetBytes("0000000`0").CopyTo($header,108)
        [Text.Encoding]::ASCII.GetBytes("0000000`0").CopyTo($header,116)
        [Text.Encoding]::ASCII.GetBytes("00000000000`0").CopyTo($header,124)
        [Text.Encoding]::ASCII.GetBytes("00000000000`0").CopyTo($header,136)
        for ($i=148; $i -lt 156; $i++) { $header[$i] = 32 }
        $header[156] = 48
        [Text.Encoding]::ASCII.GetBytes("ustar`000").CopyTo($header,257)
        $sum = 0; foreach ($value in $header) { $sum += $value }
        [Text.Encoding]::ASCII.GetBytes(([Convert]::ToString($sum,8).PadLeft(6,'0') + "`0 ")).CopyTo($header,148)
        $raw.Write($header,0,512)
    }
    $raw.Write((New-Object byte[] 1024),0,1024)
    $compressed = New-Object IO.MemoryStream
    $gzip = New-Object IO.Compression.GZipStream($compressed,[IO.Compression.CompressionMode]::Compress,$true)
    $bytes = $raw.ToArray(); $gzip.Write($bytes,0,$bytes.Length); $gzip.Dispose(); $raw.Dispose()
    $result = $compressed.ToArray(); $compressed.Dispose()
    return ,$result
}
function Assert-Refusal([string]$Name,[string]$Expected,$Transport,[switch]$Latest) {
    $testClient = New-Object Net.Http.HttpClient($Transport)
    $before = @(Get-ChildItem -LiteralPath $env:USERPROFILE -Directory -Filter 'axiom-install-*' | Select-Object -ExpandProperty FullName)
    $message = ''
    try {
        if ($Latest) { & $bootstrap -BinDir $bin -ReceiptDir $receipt }
        else { & $bootstrap -Version v1.0.0 -BinDir $bin -ReceiptDir $receipt }
    } catch { $message = $_.Exception.Message }
    if ($message -notmatch $Expected) { throw "${Name}: unexpected refusal: $message" }
    if ((Test-Path -LiteralPath $bin) -or (Test-Path -LiteralPath $receipt)) { throw "${Name}: target mutation" }
    $after = @(Get-ChildItem -LiteralPath $env:USERPROFILE -Directory -Filter 'axiom-install-*' | Select-Object -ExpandProperty FullName)
    if (@(Compare-Object $before $after).Count -ne 0) { throw "${Name}: staging leak" }
    if (@($Transport.Requests | Where-Object { $_ -notlike 'https://*' }).Count) { throw 'HTTP downgrade reached transport.' }
    Write-Output "bootstrap_refusal=$Name"
}
try {
    # Keep production staging and cleanup inside this isolated test profile.
    $env:USERPROFILE = $work
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'; $env:PROCESSOR_ARCHITEW6432 = ''
    $transport = New-Object BootstrapTransport
    $transport.Redirect = 'http://example.invalid/release'
    Assert-Refusal 'https-downgrade' 'Only HTTPS' $transport
    if ($transport.Requests.Count -ne 1) { throw 'Redirect was followed before validation.' }
    foreach ($case in @('missing','duplicate','tampered')) {
        $transport = New-Object BootstrapTransport
        $entry = ('0'*64) + "  $asset`n"
        $sums = switch ($case) { missing { '' } duplicate { $entry+$entry } tampered { $entry } }
        $transport.Files[$base+'SHA256SUMS'] = $utf8.GetBytes($sums)
        $transport.Files[$base+$asset] = $utf8.GetBytes('tampered')
        $expected = if ($case -eq 'tampered') { 'checksum mismatch' } else { 'checksum is unavailable' }
        Assert-Refusal $case $expected $transport
    }
    foreach ($path in @('../escape','/absolute','C:/absolute','axiom-1.0.0-windows-amd64/../escape','axiom-1.0.0-windows-amd64/file:stream','axiom-1.0.0-windows-amd64/back\slash')) {
        $transport = New-Object BootstrapTransport
        $archive = New-ArchiveBytes @('axiom-1.0.0-windows-amd64/axiom.exe',$path)
        $sha = [Security.Cryptography.SHA256]::Create()
        $hash = [BitConverter]::ToString($sha.ComputeHash($archive)).Replace('-','').ToLowerInvariant(); $sha.Dispose()
        $transport.Files[$base+'SHA256SUMS'] = $utf8.GetBytes("$hash  $asset`n")
        $transport.Files[$base+$asset] = $archive
        Assert-Refusal "archive-$path" 'Invalid archive root|Unsafe archive path|Invalid release archive' $transport
    }
    foreach ($json in @('{"tag_name":"invalid","draft":false,"prerelease":false}', '{"tag_name":"v1.0.0","draft":true,"prerelease":false}', '{"tag_name":"v1.0.0-rc.1","draft":false,"prerelease":true}', '{"tag_name":"v1.0.0"}', '{"tag_name":"v1.0.0","draft":"false","prerelease":false}', '[]')) {
        $transport = New-Object BootstrapTransport
        $transport.Files['https://api.github.com/repos/rgomids/axiom/releases/latest'] = $utf8.GetBytes($json)
        Assert-Refusal 'release-metadata' 'Invalid release tag|No stable release|Invalid release metadata' $transport -Latest
    }
    $env:PROCESSOR_ARCHITECTURE = 'ARM64'
    Assert-Refusal 'architecture' 'Windows amd64' (New-Object BootstrapTransport)
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $script:productType = 3
    Assert-Refusal 'server' 'Windows Server is unsupported' (New-Object BootstrapTransport)
    Write-Output 'windows_bootstrap_contract=pass'
} finally {
    $env:PROCESSOR_ARCHITECTURE = $savedArch; $env:PROCESSOR_ARCHITEW6432 = $savedWow
    $env:USERPROFILE = $savedProfile
    if (-not [IO.Path]::GetFullPath($work).StartsWith([IO.Path]::GetFullPath($testRoot) + '\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Test cleanup escaped its root.' }
    Remove-Item -LiteralPath $work -Recurse -Force
}
