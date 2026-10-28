$ErrorActionPreference = 'Stop'

# The blob webhook key is created only after the function code is indexed.
$key = $null
for ($attempt = 0; $attempt -lt 30; $attempt++) {
    $key = az functionapp keys list --subscription $env:AZURE_SUBSCRIPTION_ID `
        --resource-group $env:AZURE_RESOURCE_GROUP --name $env:AZURE_FUNCTION_NAME `
        --query 'systemKeys.blobs_extension' --output tsv --only-show-errors 2>$null
    if ($LASTEXITCODE -eq 0 -and $key -and $key -ne 'null') { break }
    Start-Sleep -Seconds 10
}
if (-not $key -or $key -eq 'null') { throw 'Blob extension key unavailable after deployment.' }

# Resolve the webhook key inside ARM, not in command-line arguments or outputs.
az deployment group create --subscription $env:AZURE_SUBSCRIPTION_ID `
    --resource-group $env:AZURE_RESOURCE_GROUP --name blob-event-subscription `
    --template-file (Join-Path $PSScriptRoot '..\infra\eventgrid.bicep') `
    --parameters "functionAppName=$env:AZURE_FUNCTION_NAME" "storageAccountName=$env:AZURE_STORAGE_ACCOUNT_NAME" `
    --output none --only-show-errors
if ($LASTEXITCODE -ne 0) { throw 'Could not create the BlobCreated event subscription.' }
Write-Host 'BlobCreated events are connected to processBlob.'
