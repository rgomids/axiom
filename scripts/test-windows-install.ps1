#requires -Version 5.1
# Offline native installer contract. No release, network or user installation.
$ErrorActionPreference = 'Stop'
$repository = Split-Path $PSScriptRoot -Parent
$work = Join-Path ([IO.Path]::GetTempPath()) ('axiom-windows-test-' + [guid]::NewGuid().ToString('N'))
$utf8 = New-Object Text.UTF8Encoding($false)
function Write-TestFile([string]$Path,[string]$Content) { [IO.File]::WriteAllText($Path,$Content,$utf8) }
function Assert-NativeExit([string]$Label) { if ($LASTEXITCODE -ne 0) { throw "$Label failed: $LASTEXITCODE" } }
function New-TestBundle([string]$Version) {
    $name = "axiom-$Version-windows-amd64"
    $root = Join-Path $work $name
    New-Item -ItemType Directory -Path $root | Out-Null
    & go build -trimpath -ldflags "-X main.buildVersion=$Version -X main.buildRevision=123456789abc -X main.buildSourceState=clean -X main.buildRelease=true" -o (Join-Path $root 'axiom.exe') ./cmd/lingo
    Assert-NativeExit 'Build Windows executable'
    Copy-Item -LiteralPath (Join-Path $repository 'LICENSE') -Destination $root
    Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'install-release.ps1') -Destination (Join-Path $root 'install.ps1')
    Copy-Item -LiteralPath (Join-Path $repository 'internal\codexruntime\skills') -Destination $root -Recurse
    $skillManifest = "formatVersion=1`nskillSetVersion=1`nbinaryCompatibility=1`n"
    foreach ($skill in (Get-ChildItem -LiteralPath (Join-Path $root 'skills') -Directory | Sort-Object Name)) {
        $hash = (Get-FileHash -LiteralPath (Join-Path $skill.FullName 'SKILL.md')).Hash.ToLowerInvariant()
        $skillManifest += "skill.$($skill.Name)=$hash`n"
    }
    Write-TestFile (Join-Path $root 'skills-manifest.txt') $skillManifest
    Write-TestFile (Join-Path $root 'release-metadata.txt') "formatVersion=1`nproduct=Axiom`nversion=$Version`nrevision=123456789abc`nsourceState=clean`nrelease=true`nplatform=windows`ngoos=windows`narchitecture=amd64`nskillSetVersion=1`n"
    $manifest = ''
    foreach ($file in (Get-ChildItem -LiteralPath $root -Recurse -File | Sort-Object FullName)) {
        $relative = $file.FullName.Substring($root.Length+1).Replace('\','/')
        $hash = (Get-FileHash -LiteralPath $file.FullName).Hash.ToLowerInvariant()
        $manifest += "$hash  $relative`n"
    }
    Write-TestFile (Join-Path $root 'MANIFEST.sha256') $manifest
    $archive = Join-Path $work "$name.tar.gz"
    & tar.exe -czf $archive -C $work $name
    Assert-NativeExit 'Create Windows archive'
    $sums = Join-Path $work "$Version-SHA256SUMS"
    $hash = (Get-FileHash -LiteralPath $archive).Hash.ToLowerInvariant()
    Write-TestFile $sums "$hash  $name.tar.gz`n"
    return @{ Root=$root; Archive=$archive; Checksums=$sums }
}
function Install-TestBundle($Bundle) {
    & (Join-Path $Bundle.Root 'install.ps1') -Archive $Bundle.Archive -Checksums $Bundle.Checksums -BinDir $bin -ReceiptDir $receipt
}
New-Item -ItemType Directory -Path $work | Out-Null
$savedEnvironment = @{}
foreach ($name in @('LINGO_PROJECTS_ROOT','LINGO_STATE_ROOT','AXIOM_CODEX_SKILLS_ROOT','LOCALAPPDATA','USERPROFILE','CLAUDE_CONFIG_DIR','PATH')) {
    $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name,'Process')
    if ($name -ne 'PATH') { [Environment]::SetEnvironmentVariable($name,(Join-Path $work $name),'Process') }
}
Push-Location $repository
try {
    $bin = Join-Path $work 'installation with spaces\bin'
    $receipt = Join-Path $work 'installation with spaces\receipts'
    $first = New-TestBundle '1.0.0'
    $bad = Join-Path $work 'bad-checksums'
    Write-TestFile $bad (('0'*64) + '  ' + [IO.Path]::GetFileName($first.Archive) + "`n")
    $rejected = $false
    try { & (Join-Path $first.Root 'install.ps1') -Archive $first.Archive -Checksums $bad -BinDir $bin -ReceiptDir $receipt } catch { $rejected = $true }
    if (-not $rejected -or (Test-Path -LiteralPath $bin) -or (Test-Path -LiteralPath $receipt)) { throw 'Checksum refusal must have zero installation effects.' }
    if ((Get-CimInstance Win32_OperatingSystem).ProductType -ne 1) {
        # Windows PowerShell represents native stderr as error records; capture
        # this expected failure without terminating before checking its exit code.
        $savedPreference = $ErrorActionPreference
        try {
            $ErrorActionPreference = 'Continue'
            $output = & (Join-Path $first.Root 'axiom.exe') install-release --archive $first.Archive --checksums $first.Checksums --bin-dir $bin --receipt-dir $receipt 2>&1
        } finally { $ErrorActionPreference = $savedPreference }
        if ($LASTEXITCODE -eq 0 -or ($output -join "`n") -notmatch 'unsupported_host' -or (Test-Path -LiteralPath $bin) -or (Test-Path -LiteralPath $receipt)) {
            throw 'Windows Server must be refused without target mutation.'
        }
        Write-Output 'windows_server_refusal=pass; client_install_acceptance=not_run'
        # The expected native refusal must not become the CI shell's exit code.
        exit 0
    }
    Install-TestBundle $first
    # Full default flow in a synthetic profile: AppData is intentionally unsafe,
    # Runtime roots require explicit repair, and unrelated skills are preserved.
    $isolatedOverrides = @{}
    foreach ($name in @('LINGO_PROJECTS_ROOT','LINGO_STATE_ROOT','AXIOM_CODEX_SKILLS_ROOT')) {
        $isolatedOverrides[$name] = [Environment]::GetEnvironmentVariable($name,'Process')
        [Environment]::SetEnvironmentVariable($name,$null,'Process')
    }
    New-Item -ItemType Directory -Path $env:USERPROFILE,$env:LOCALAPPDATA -Force | Out-Null
    $everyone = New-Object Security.Principal.SecurityIdentifier('S-1-1-0')
    $unsafeRule = New-Object Security.AccessControl.FileSystemAccessRule($everyone,'FullControl','ContainerInherit,ObjectInherit','None','Allow')
    $appDataACL = Get-Acl -LiteralPath $env:LOCALAPPDATA
    $appDataACL.AddAccessRule($unsafeRule)
    Set-Acl -LiteralPath $env:LOCALAPPDATA -AclObject $appDataACL
    $appDataBefore = (Get-Acl -LiteralPath $env:LOCALAPPDATA).Sddl
    $skills = Join-Path $env:USERPROFILE '.agents\skills'
    $other = Join-Path $skills 'unrelated\SKILL.md'
    New-Item -ItemType Directory -Path (Split-Path $other -Parent) -Force | Out-Null
    Write-TestFile $other 'unrelated skill preserved'
    $agents = Join-Path $env:USERPROFILE '.agents'
    $agentsACL = Get-Acl -LiteralPath $agents
    $agentsACL.AddAccessRule($unsafeRule)
    Set-Acl -LiteralPath $agents -AclObject $agentsACL
    $otherBefore = (Get-Acl -LiteralPath $other).Sddl
    $defaultBin = Join-Path $env:USERPROFILE '.axiom\windows\bin'
    $defaultReceipt = Join-Path $env:USERPROFILE '.axiom\windows\install'
    $savedPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $preview = 'no' | & (Join-Path $first.Root 'axiom.exe') install-release --archive $first.Archive --checksums $first.Checksums --bin-dir $defaultBin --receipt-dir $defaultReceipt 2>&1
    } finally { $ErrorActionPreference = $savedPreference }
    if ($LASTEXITCODE -eq 0 -or (Test-Path -LiteralPath $defaultBin)) { throw 'Declined repair published an installation.' }
    if (($preview -join "`n") -notmatch '\[S/n\]' -or ($preview -join "`n") -match 'permission_before=|permission_after=|Type REPAIR') { throw "Unexpected permission prompt: $preview" }
    's' | & (Join-Path $first.Root 'axiom.exe') install-release --archive $first.Archive --checksums $first.Checksums --bin-dir $defaultBin --receipt-dir $defaultReceipt
    Assert-NativeExit 'Approved default onboarding repair'
    if ((Get-Acl -LiteralPath $other).Sddl -cne $otherBefore -or
        (Get-Acl -LiteralPath $env:LOCALAPPDATA).Sddl -cne $appDataBefore -or
        [IO.File]::ReadAllText($other) -cne 'unrelated skill preserved') { throw 'Repair changed unrelated data or AppData.' }
    $runtimeBin = Join-Path $work 'fake-runtime'
    New-Item -ItemType Directory -Path $runtimeBin | Out-Null
    Write-TestFile (Join-Path $runtimeBin 'codex.cmd') '@echo runtime-must-not-execute'
    $env:PATH = "$runtimeBin;$env:PATH"
    & (Join-Path $defaultBin 'axiom.exe') first-run
    Assert-NativeExit 'Default Runtime setup'
    if (-not (Test-Path -LiteralPath (Join-Path $skills 'axiom-project\SKILL.md'))) { throw 'Codex skill discovery path missing.' }
    $beforeReinstall = (Get-FileHash -LiteralPath (Join-Path $defaultReceipt 'installation.receipt')).Hash
    & (Join-Path $first.Root 'install.ps1') -Archive $first.Archive -Checksums $first.Checksums
    Assert-NativeExit 'Default Windows paths'
    if (-not (Test-Path (Join-Path $env:USERPROFILE '.axiom\windows\bin\axiom.exe')) -or
        -not (Test-Path (Join-Path $env:USERPROFILE '.axiom\windows\install\installation.receipt'))) { throw 'Default installation missing.' }
    Write-Output 'windows_default_paths=pass'
    if ((Get-FileHash -LiteralPath (Join-Path $defaultReceipt 'installation.receipt')).Hash -cne $beforeReinstall) { throw 'Default reinstall changed receipt.' }
    & (Join-Path $defaultBin 'axiom.exe') version
    Assert-NativeExit 'Default version command'
    Write-Output 'windows_default_onboarding=pass; unsafe_appdata_preserved=pass; consent=pass; codex_discovery=pass; reinstall=pass'
    foreach ($name in $isolatedOverrides.Keys) { [Environment]::SetEnvironmentVariable($name,$isolatedOverrides[$name],'Process') }
    $env:PATH = $savedEnvironment['PATH']

    $unsafe = Join-Path $work 'unsafe ancestor'
    New-Item -ItemType Directory -Path $unsafe | Out-Null
    $acl = Get-Acl -LiteralPath $unsafe
    $everyone = New-Object Security.Principal.SecurityIdentifier('S-1-1-0')
    $rule = New-Object Security.AccessControl.FileSystemAccessRule($everyone,'DeleteSubdirectoriesAndFiles','None','None','Allow')
    $acl.AddAccessRule($rule)
    Set-Acl -LiteralPath $unsafe -AclObject $acl
    $beforeACL = (Get-Acl -LiteralPath $unsafe).Sddl
    $savedPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $output = & (Join-Path $first.Root 'axiom.exe') install-release --archive $first.Archive --checksums $first.Checksums --bin-dir (Join-Path $unsafe 'bin') --receipt-dir (Join-Path $unsafe 'receipts') 2>&1
    } finally { $ErrorActionPreference = $savedPreference }
    if ($LASTEXITCODE -eq 0 -or ($output -join "`n") -notmatch 'rule=ancestors must prevent replacement' -or
        ($output -join "`n") -notmatch 'unsafe ancestor' -or ($output -join "`n") -notmatch 'install_next:.*-BinDir.*-ReceiptDir' -or
        (Test-Path (Join-Path $unsafe 'receipts')) -or (Test-Path (Join-Path $unsafe 'bin')) -or
        (Get-Acl -LiteralPath $unsafe).Sddl -cne $beforeACL) { throw "Unsafe ancestor diagnostic/preservation failed: $output" }
    Write-Output 'windows_storage_diagnostic=pass'
    # Existing owned installs must retain the same actionable cause through
    # the upgrade preview, while preserving the receipt and executable.
    $defaultBin = Join-Path $env:USERPROFILE '.axiom\windows\bin'
    $defaultReceipt = Join-Path $env:USERPROFILE '.axiom\windows\install'
    $beforeBinary = (Get-FileHash (Join-Path $defaultBin 'axiom.exe')).Hash
    $beforeReceipt = (Get-FileHash (Join-Path $defaultReceipt 'installation.receipt')).Hash
    # This profile is a fixture created above, never the real user's profile.
    $defaultParent = $env:USERPROFILE
    $acl = New-Object Security.AccessControl.DirectorySecurity
    $acl.SetSecurityDescriptorSddlForm((Get-Acl -LiteralPath $defaultParent).Sddl,[Security.AccessControl.AccessControlSections]::Access)
    $acl.AddAccessRule($rule)
    Set-Acl -LiteralPath $defaultParent -AclObject $acl
    try {
        $ErrorActionPreference = 'Continue'
        $output = & (Join-Path $first.Root 'axiom.exe') install-release --archive $first.Archive --checksums $first.Checksums --bin-dir $defaultBin --receipt-dir $defaultReceipt 2>&1
    } finally { $ErrorActionPreference = $savedPreference }
    if ($LASTEXITCODE -eq 0 -or ($output -join "`n") -notmatch 'rule=ancestors' -or
        ($output -join "`n") -notmatch 'onboarding_root=' -or
        (Get-FileHash (Join-Path $defaultBin 'axiom.exe')).Hash -cne $beforeBinary -or
        (Get-FileHash (Join-Path $defaultReceipt 'installation.receipt')).Hash -cne $beforeReceipt) { throw "Upgrade storage diagnostic/preservation failed: $output" }
    Write-Output 'windows_upgrade_storage_diagnostic=pass'
    $receiptFile = Join-Path $receipt 'installation.receipt'
    $before = [IO.File]::ReadAllText($receiptFile)
    Install-TestBundle $first
    if ([IO.File]::ReadAllText($receiptFile) -cne $before) { throw 'Reinstall changed the receipt.' }
    & (Join-Path $bin 'axiom.exe') version
    Assert-NativeExit 'Installed executable version'
    $next = New-TestBundle '1.1.0'
    $held = [IO.File]::Open((Join-Path $bin 'axiom.exe'),[IO.FileMode]::Open,[IO.FileAccess]::Read,[IO.FileShare]::None)
    try {
        $rejected = $false
        try { Install-TestBundle $next } catch { $rejected = $true }
        if (-not $rejected -or [IO.File]::ReadAllText($receiptFile) -cne $before) { throw 'An in-use binary was not preserved.' }
    } finally { $held.Dispose() }
    Install-TestBundle $next
    $version = & (Join-Path $bin 'axiom.exe') version
    Assert-NativeExit 'Upgraded executable version'
    if (($version -join "`n") -notmatch '1\.1\.0') { throw 'Upgrade did not publish the new executable.' }
    foreach ($arguments in @(@('-Version','invalid'),@('-Version','v1.0.0','-Channel','stable'))) {
        $rejected = $false
        try { & (Join-Path $PSScriptRoot 'install.ps1') @arguments } catch { $rejected = $true }
        if (-not $rejected) { throw 'Invalid bootstrap selector accepted.' }
    }
    Write-Output 'windows_install_contract=pass'
} finally {
    Pop-Location
    foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name,$savedEnvironment[$name],'Process') }
    # Exact UUID directory created above; never delete a caller-supplied path.
    if (Test-Path -LiteralPath $work) { Remove-Item -LiteralPath $work -Recurse -Force }
}
