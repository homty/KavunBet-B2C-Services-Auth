$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot

function Get-WireContract {
    param([Parameter(Mandatory)][string]$Path)

    Get-Content -LiteralPath $Path |
        Where-Object { $_ -notmatch '^\s*option\s+go_package\s*=' } |
        ForEach-Object { $_.TrimEnd() }
}

function Assert-MatchingContract {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$ProviderPath,
        [Parameter(Mandatory)][string]$ConsumerPath
    )

    $providerContract = Get-WireContract -Path $ProviderPath
    $consumerContract = Get-WireContract -Path $ConsumerPath
    $difference = Compare-Object $providerContract $consumerContract -SyncWindow 0

    if ($difference) {
        throw "$Name contract drift detected between provider and auth-service consumer copies."
    }

    Write-Host "$Name contract matches."
}

Assert-MatchingContract `
    -Name "user.v1" `
    -ProviderPath (Join-Path $repositoryRoot "services/user-service/api/user/v1/user.proto") `
    -ConsumerPath (Join-Path $repositoryRoot "services/auth-service/internal/contracts/user/v1/user.proto")

Assert-MatchingContract `
    -Name "wallet.v1" `
    -ProviderPath (Join-Path $repositoryRoot "services/wallet-service/api/wallet/v1/wallet.proto") `
    -ConsumerPath (Join-Path $repositoryRoot "services/auth-service/internal/contracts/wallet/v1/wallet.proto")

Write-Host "All protobuf wire contracts match."
