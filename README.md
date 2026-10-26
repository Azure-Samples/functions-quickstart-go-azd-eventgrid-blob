<!--
---
name: Azure Functions Go Event Grid Blob Trigger using Azure Developer CLI
description: Stream blobs from an Event Grid-based Blob Storage trigger in Go on Azure Functions Flex Consumption.
page_type: sample
languages:
- azdeveloper
- go
products:
- azure
- azure-functions
- azure-blob-storage
urlFragment: functions-quickstart-go-azd-eventgrid-blob
---
-->

# Azure Functions Go Event Grid Blob Trigger

This sample follows the [Go timer quickstart](https://github.com/Azure-Samples/functions-quickstart-go-azd-timer)
and deploys a Go function to Linux Flex Consumption using Azure Developer CLI
(`azd`). Go worker packages are currently preview packages.

Uploading a blob to `samples-workitems` raises a `Microsoft.Storage.BlobCreated`
event. Event Grid calls the Blob extension webhook, and `processBlob` receives
an authenticated `*blob.Client`. The handler streams the blob into a SHA-256
hash and logs the byte count and digest without logging its content.
It returns download/read errors so the trigger can retry.

## Prerequisites

- [Go 1.25 or later](https://go.dev/dl/)
- [Azure Functions Core Tools 4.12 or later](https://learn.microsoft.com/azure/azure-functions/functions-run-local)
- [Azure CLI](https://learn.microsoft.com/cli/azure/install-azure-cli)
- [Azure Developer CLI](https://learn.microsoft.com/azure/developer/azure-developer-cli/install-azd)
- [PowerShell 7](https://learn.microsoft.com/powershell/scripting/install/installing-powershell) for the cross-platform postdeploy hook
- An Azure subscription and permissions to create resources and role assignments
- Optional: VS Code with the Azure Functions and Go extensions; Azurite for local testing

## Initialize

```shell
git clone https://github.com/Azure-Samples/functions-quickstart-go-azd-eventgrid-blob.git
cd functions-quickstart-go-azd-eventgrid-blob
```

## Run locally

```powershell
Copy-Item local.settings.example.json local.settings.json
```

Start Azurite in a separate terminal, create a `samples-workitems` container
in the emulator, and upload a test file using Azure Storage Explorer.
Then run `func start`. The host builds the Go binary and indexes `processBlob`.

Azurite does not emit Event Grid events. To simulate delivery locally, use
**Azure Functions: Execute Function Now** in VS Code and enter
`samples-workitems/<uploaded-file-name>` as the request body. Alternatively,
use the Azure deployment below to exercise actual Event Grid delivery.

Triggers are registered in [`main.go`](./main.go); there is no `function.json`.
The blank import of `triggers/blob` enables SDK-client injection.

## Deploy

```shell
az login
azd auth login
azd up
```

If your azd version skips lifecycle hooks in noninteractive mode, explicitly run
`azd hooks run postdeploy --service api` before uploading a test blob.

The infrastructure includes managed identity, keyless host storage, Application
Insights, and a VNet/private endpoints by default. The identity has Blob Data
Owner and Queue Data Contributor permissions for blob processing, receipts,
and poison messages.

The postdeploy hook in [`scripts/subscribe.ps1`](./scripts/subscribe.ps1) waits
for the Blob extension key and creates the filtered Event Grid subscription.
It runs after deployment because the webhook is unavailable before indexing.
The webhook requires its system key even though storage access is keyless.
The key is not written to source or printed by the script.

For testing uploads from your workstation, disable the storage network isolation
**before provisioning**, and provision a data-plane role for your signed-in user:

```powershell
azd env new blob-dev
azd env set VNET_ENABLED false
azd env set AZURE_PRINCIPAL_ID (az ad signed-in-user show --query id -o tsv)
azd up
```

Keep the default VNet configuration when testing from a network that can reach
the storage private endpoints. Turning off network isolation does not enable
anonymous blob access or shared-key authentication.

## Verify in Azure

Load the deployment outputs and upload a file:

```powershell
$deployment = azd env get-values --output json | ConvertFrom-Json
az storage blob upload --auth-mode login `
  --account-name $deployment.AZURE_STORAGE_ACCOUNT_NAME `
  --container-name samples-workitems --name sample.go --file main.go --overwrite
```

Allow a few minutes for role assignments to propagate on the first deployment.
In the function app's Application Insights **Logs**, run:

```kusto
traces
| where timestamp > ago(15m)
| where message contains "blob processed"
| project timestamp, message, customDimensions
```

The log includes `bytes` and `sha256`. Uploads outside `samples-workitems`
do not match the event subscription. Blob triggers can deliver an event more
than once; use idempotent processing when replacing this demonstration with
business side effects.

## Tests and redeployment

```shell
go test ./...
go vet ./...
azd up
```

Tests cover empty and large blob downloads, download failures, and Event Grid
trigger registration. Azure end-to-end verification requires a real storage
upload; unit tests do not simulate Event Grid itself.

## Clean up

```shell
azd down
```
