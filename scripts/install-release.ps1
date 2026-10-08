#requires -Version 5.1
[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$Archive,
    [Parameter(Mandatory=$true)][string]$Checksums,
    [string]$BinDir = (Join-Path $env:USERPROFILE '.axiom\windows\bin'),
    [string]$ReceiptDir = (Join-Path $env:USERPROFILE '.axiom\windows\install')
)
$ErrorActionPreference = 'Stop'
$legacyBin = Join-Path $env:LOCALAPPDATA 'Axiom\bin'
$legacyReceipt = Join-Path $env:LOCALAPPDATA 'Axiom\install'
if (-not ((Test-Path -LiteralPath (Join-Path $env:USERPROFILE '.axiom\windows\bin\axiom.exe')) -or
          (Test-Path -LiteralPath (Join-Path $env:USERPROFILE '.axiom\windows\install\installation.receipt'))) -and
    ((Test-Path -LiteralPath (Join-Path $legacyBin 'axiom.exe')) -or
     (Test-Path -LiteralPath (Join-Path $legacyReceipt 'installation.receipt')))) {
    if (-not $PSBoundParameters.ContainsKey('BinDir')) { $BinDir = $legacyBin }
    if (-not $PSBoundParameters.ContainsKey('ReceiptDir')) { $ReceiptDir = $legacyReceipt }
}
& (Join-Path $PSScriptRoot 'axiom.exe') install-release --archive $Archive --checksums $Checksums --bin-dir $BinDir --receipt-dir $ReceiptDir
if ($LASTEXITCODE -ne 0) { throw "Verified release installation failed (exit $LASTEXITCODE)." }
