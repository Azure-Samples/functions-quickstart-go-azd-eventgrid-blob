param functionAppName string
param storageAccountName string

resource functionApp 'Microsoft.Web/sites@2024-11-01' existing = {
  name: functionAppName
}

resource storage 'Microsoft.Storage/storageAccounts@2023-05-01' existing = {
  name: storageAccountName
}

var key = listKeys('${functionApp.id}/host/default', '2024-11-01').systemKeys.blobs_extension

resource subscription 'Microsoft.EventGrid/eventSubscriptions@2022-06-15' = {
  name: 'blob-created'
  scope: storage
  properties: {
    destination: {
      endpointType: 'WebHook'
      properties: {
        endpointUrl: 'https://${functionApp.properties.defaultHostName}/runtime/webhooks/blobs?functionName=Host.Functions.processBlob&code=${uriComponent(key)}'
      }
    }
    filter: {
      includedEventTypes: ['Microsoft.Storage.BlobCreated']
      subjectBeginsWith: '/blobServices/default/containers/samples-workitems/blobs/'
    }
    retryPolicy: {
      maxDeliveryAttempts: 30
      eventTimeToLiveInMinutes: 1440
    }
  }
}
