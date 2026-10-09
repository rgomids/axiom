#requires -Version 5.1
# Execute the production bootstrap with an in-memory HTTP transport. Host data
# is injected only in this test; production exposes no support-policy override.
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http
$references = @('System.Net.Http')
if ($PSVersionTable.PSEdition -eq 'Core') { $references += @('System.Collections','System.Runtime','System.Net.Primitives') }
Add-Type -ReferencedAssemblies $references -TypeDefinition @'
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
# Replace registry and broadcast boundaries as well: tests never write the
# real user's environment or notify other applications.
class BootstrapUserEnvironment {
    [object]$Value = '%SystemRoot%\System32;C:\Existing;;'
    [Microsoft.Win32.RegistryValueKind]$Kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
    [int]$Writes = 0
    [int]$Reads = 0
    [bool]$Race = $false
    [object] GetValue([string]$name,[object]$fallback,[Microsoft.Win32.RegistryValueOptions]$options) {
        $this.Reads++
        if ($this.Race -and $this.Reads -eq 2) { $this.Value = 'C:\Concurrent' }
        return $this.Value
    }
    [Microsoft.Win32.RegistryValueKind] GetValueKind([string]$name) { return $this.Kind }
    [void] SetValue([string]$name,[object]$value,[Microsoft.Win32.RegistryValueKind]$kind) { $this.Value=$value; $this.Kind=$kind; $this.Writes++ }
    [void] Dispose() {}
}
$testUserEnvironment = [BootstrapUserEnvironment]::new()
$registryConstruction = "[Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')"
if (-not $source.Contains($registryConstruction)) { throw 'Bootstrap user environment seam changed.' }
$testSource = $source.Replace($construction, '$client = $testClient').Replace($registryConstruction, '$testUserEnvironment').Replace('[AxiomEnvironmentNotification]::Notify()', '# Test: no global environment broadcast')
$bootstrap = [scriptblock]::Create($testSource)
# Exercise registry refusal boundaries independently of release transport.
$tokens=$null; $parseErrors=$null
$ast=[Management.Automation.Language.Parser]::ParseInput($testSource,[ref]$tokens,[ref]$parseErrors)
if ($parseErrors.Count) { throw 'Bootstrap parse failure.' }
$pathFunction=$ast.Find({ param($node) $node -is [Management.Automation.Language.FunctionDefinitionAst] -and $node.Name -eq 'Update-AxiomUserPath' },$true)
. ([scriptblock]::Create($pathFunction.Extent.Text))
foreach ($case in @('missing','expanded-duplicate','unsupported','length','race')) {
    $testUserEnvironment=[BootstrapUserEnvironment]::new()
    $directory='C:\Axiom\bin'
    switch ($case) {
        missing { $testUserEnvironment.Value=$null }
        expanded-duplicate { $directory=Join-Path $env:USERPROFILE 'Axiom\bin'; $testUserEnvironment.Value='%USERPROFILE%\Axiom\bin' }
        unsupported { $testUserEnvironment.Kind=[Microsoft.Win32.RegistryValueKind]::MultiString; $testUserEnvironment.Value=@('C:\Existing') }
        length { $testUserEnvironment.Value='x'*32766 }
        race { $testUserEnvironment.Race=$true }
    }
    $message=''
    try { Update-AxiomUserPath $directory } catch { $message=$_.Exception.Message }
    if ($case -eq 'missing') {
        if ($message -or $testUserEnvironment.Value -cne $directory -or $testUserEnvironment.Writes -ne 1) { throw 'Missing user PATH setup failed.' }
    } elseif ($case -eq 'expanded-duplicate') {
        if ($message -or $testUserEnvironment.Writes) { throw 'Expandable PATH entry was duplicated.' }
    } else {
        if (-not $message -or $testUserEnvironment.Writes) { throw "User PATH refusal failed: $case" }
        if ($case -eq 'race' -and $testUserEnvironment.Value -cne 'C:\Concurrent') { throw 'Concurrent user PATH was overwritten.' }
    }
}
$testUserEnvironment=[BootstrapUserEnvironment]::new()
Write-Output 'windows_user_path_boundaries=pass'
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
$onboardingEnvironment = @{}
foreach ($name in @('LOCALAPPDATA','LINGO_PROJECTS_ROOT','LINGO_STATE_ROOT','AXIOM_CODEX_SKILLS_ROOT','CLAUDE_CONFIG_DIR','PATH')) {
    $onboardingEnvironment[$name] = [Environment]::GetEnvironmentVariable($name,'Process')
}
$script:productType = 1
$script:osVersion = '10.0.22621'
function Get-CimInstance { param($ClassName) [pscustomobject]@{ProductType=$script:productType;Version=$script:osVersion} }

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
    # Issue #189: exercise IEX itself, including a caller variable that is not
    # a valid selector. A parameter declaration in IEX's caller scope failed
    # even without a pre-existing Channel. No network or mutation is needed.
    $script:productType = 2
    foreach ($callerChannel in @('', 'caller-value')) {
        $Channel = $callerChannel
        $message = ''
        try { $source | Invoke-Expression } catch { $message = $_.Exception.Message }
        if ($message -notmatch 'domain controllers and unknown product types are unsupported' -or $Channel -cne $callerChannel) {
            throw "IEX child scope regression: $message"
        }
    }
    Remove-Variable Channel
    $message = ''
    try { $source | Invoke-Expression } catch { $message = $_.Exception.Message }
    if ($message -notmatch 'domain controllers and unknown product types are unsupported') { throw "IEX omitted Channel regression: $message" }
    $script:productType = 1
    Write-Output 'bootstrap_iex_scope=pass'
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
    # Issue #183: the numeric OS version never decides eligibility. A client
    # host of any version passes host checks and reaches the network stage.
    foreach ($clientVersion in @('10.0.17134','10.0.17763','10.0.26100','10.1.0','11.0.0','6.3.9600')) {
        $script:osVersion = $clientVersion
        $transport = New-Object BootstrapTransport
        $transport.Redirect = 'http://example.invalid/release'
        Assert-Refusal "client-version-$clientVersion" 'Only HTTPS' $transport
        if ($transport.Requests.Count -ne 1) { throw "client-version-${clientVersion}: host check refused before network" }
    }
    $script:osVersion = '10.0.22621'
    $script:productType = 3
    foreach ($serverVersion in @('10.0.20348','10.0.17763')) {
        $script:osVersion = $serverVersion
        $transport = New-Object BootstrapTransport
        $transport.Redirect = 'http://example.invalid/release'
        Assert-Refusal "server-version-$serverVersion" 'Only HTTPS' $transport
        if ($transport.Requests.Count -ne 1) { throw 'Server refused before transport.' }
    }
    foreach ($unsupportedProduct in @(2,0)) {
        $script:productType = $unsupportedProduct
        $transport = New-Object BootstrapTransport
        Assert-Refusal "unsupported-product-$unsupportedProduct" 'domain controllers and unknown product types are unsupported' $transport
        if ($transport.Requests.Count -ne 0) { throw 'Unsupported product reached transport.' }
    }
    # Positive transport test uses a real verified native bundle, with only
    # HTTP and host-metadata lookup replaced. Never install into the host profile.
    if ((CimCmdlets\Get-CimInstance Win32_OperatingSystem).ProductType -in @(1,3)) {
        $script:productType = (CimCmdlets\Get-CimInstance Win32_OperatingSystem).ProductType
        $repository = Split-Path $PSScriptRoot -Parent
        $bundleName = 'axiom-1.0.0-windows-amd64'
        $bundleRoot = Join-Path $work $bundleName
        New-Item -ItemType Directory -Path $bundleRoot | Out-Null
        Push-Location $repository
        try {
            & go build -trimpath -ldflags '-X main.buildVersion=1.0.0 -X main.buildRevision=123456789abc -X main.buildSourceState=clean -X main.buildRelease=true' -o (Join-Path $bundleRoot 'axiom.exe') ./cmd/lingo
            if ($LASTEXITCODE -ne 0) { throw 'Positive bootstrap fixture build failed.' }
        } finally { Pop-Location }
        Copy-Item -LiteralPath (Join-Path $repository 'LICENSE') -Destination $bundleRoot
        Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'install-release.ps1') -Destination (Join-Path $bundleRoot 'install.ps1')
        Copy-Item -LiteralPath (Join-Path $repository 'internal\codexruntime\skills') -Destination $bundleRoot -Recurse
        $skillManifest = "formatVersion=1`nskillSetVersion=1`nbinaryCompatibility=1`n"
        foreach ($skill in (Get-ChildItem -LiteralPath (Join-Path $bundleRoot 'skills') -Directory | Sort-Object Name)) {
            $skillManifest += "skill.$($skill.Name)=$((Get-FileHash -LiteralPath (Join-Path $skill.FullName 'SKILL.md')).Hash.ToLowerInvariant())`n"
        }
        [IO.File]::WriteAllText((Join-Path $bundleRoot 'skills-manifest.txt'),$skillManifest,$utf8)
        [IO.File]::WriteAllText((Join-Path $bundleRoot 'release-metadata.txt'),"formatVersion=1`nproduct=Axiom`nversion=1.0.0`nrevision=123456789abc`nsourceState=clean`nrelease=true`nplatform=windows`ngoos=windows`narchitecture=amd64`nskillSetVersion=1`n",$utf8)
        $manifest = ''
        foreach ($file in (Get-ChildItem -LiteralPath $bundleRoot -Recurse -File | Sort-Object FullName)) {
            $manifest += "$((Get-FileHash -LiteralPath $file.FullName).Hash.ToLowerInvariant())  $($file.FullName.Substring($bundleRoot.Length+1).Replace('\','/'))`n"
        }
        [IO.File]::WriteAllText((Join-Path $bundleRoot 'MANIFEST.sha256'),$manifest,$utf8)
        $archivePath = Join-Path $work $asset
        & tar.exe -czf $archivePath -C $work $bundleName
        if ($LASTEXITCODE -ne 0) { throw 'Positive bootstrap archive failed.' }
        $archiveHash = (Get-FileHash -LiteralPath $archivePath).Hash.ToLowerInvariant()
        $env:LOCALAPPDATA = Join-Path $work 'AppData\Local'
        $env:CLAUDE_CONFIG_DIR = $null
        # Modern .NET keeps an empty variable when PowerShell binds $null to
        # string. The CLI correctly refuses an explicitly empty state root.
        foreach ($name in @('LINGO_PROJECTS_ROOT','LINGO_STATE_ROOT','AXIOM_CODEX_SKILLS_ROOT')) {
            if (Test-Path -LiteralPath "Env:$name") { Remove-Item -LiteralPath "Env:$name" }
        }
        $runtimeBin = Join-Path $work 'runtime-bin'
        New-Item -ItemType Directory -Path $runtimeBin | Out-Null
        [IO.File]::WriteAllText((Join-Path $runtimeBin 'codex.cmd'),'@echo must-not-execute',$utf8)
        [IO.File]::WriteAllText((Join-Path $runtimeBin 'claude.cmd'),'@echo must-not-execute',$utf8)
        $env:PATH = "$runtimeBin;$env:PATH"
        foreach ($run in @(1,2)) {
            $transport = New-Object BootstrapTransport
            $transport.Files['https://api.github.com/repos/rgomids/axiom/releases/latest'] = $utf8.GetBytes('{"tag_name":"v1.0.0","draft":false,"prerelease":false}')
            $transport.Files[$base+'SHA256SUMS'] = $utf8.GetBytes("$archiveHash  $asset`n")
            $transport.Files[$base+$asset] = [IO.File]::ReadAllBytes($archivePath)
            $testClient = New-Object Net.Http.HttpClient($transport)
            $result = & $bootstrap
            if (($result -join "`n") -notmatch 'onboarding_status=ready' -or
                -not (Test-Path -LiteralPath (Join-Path $work '.agents\skills\axiom-project\SKILL.md')) -or
                -not (Test-Path -LiteralPath (Join-Path $work '.claude\skills\axiom-project\SKILL.md')) -or
                (Get-Command axiom -ErrorAction Stop).Source -ine (Join-Path $work '.axiom\windows\bin\axiom.exe')) { throw "Default bootstrap did not complete: $result" }
        }
        $expectedUserPath = '%SystemRoot%\System32;C:\Existing;;' + (Join-Path $work '.axiom\windows\bin')
        if ($testUserEnvironment.Value -cne $expectedUserPath -or $testUserEnvironment.Writes -ne 1 -or
            $testUserEnvironment.Kind -ne [Microsoft.Win32.RegistryValueKind]::ExpandString) { throw 'User PATH preservation/idempotence failed.' }
        Write-Output 'windows_bootstrap_default_onboarding=pass; automatic_first_run=pass; session_path=pass; user_path=pass; reinstall=pass'
    } else { throw 'Native supported Windows host required for positive bootstrap acceptance.' }
    Write-Output 'windows_bootstrap_contract=pass'
} finally {
    $env:PROCESSOR_ARCHITECTURE = $savedArch; $env:PROCESSOR_ARCHITEW6432 = $savedWow
    $env:USERPROFILE = $savedProfile
    foreach ($name in $onboardingEnvironment.Keys) {
        if ($null -eq $onboardingEnvironment[$name]) {
            if (Test-Path -LiteralPath "Env:$name") { Remove-Item -LiteralPath "Env:$name" }
        } else { [Environment]::SetEnvironmentVariable($name,$onboardingEnvironment[$name],'Process') }
    }
    if (-not [IO.Path]::GetFullPath($work).StartsWith([IO.Path]::GetFullPath($testRoot) + '\',[StringComparison]::OrdinalIgnoreCase)) { throw 'Test cleanup escaped its root.' }
    Remove-Item -LiteralPath $work -Recurse -Force
}
