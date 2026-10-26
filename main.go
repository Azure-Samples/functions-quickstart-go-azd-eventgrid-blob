package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/azure/azure-functions-golang-worker/sdk"
	_ "github.com/azure/azure-functions-golang-worker/triggers/blob"
	"github.com/azure/azure-functions-golang-worker/worker"
)

func main() {
	worker.Start(newFunctionApp())
}

func newFunctionApp() *sdk.App {
	app := sdk.FunctionApp()
	app.Blob("processBlob", processBlob,
		sdk.WithPath("samples-workitems/{name}"),
		sdk.WithConnection("AzureWebJobsStorage"),
		sdk.WithSource("EventGrid"),
	)
	return app
}

func processBlob(ctx context.Context, client *blob.Client) error {
	download, err := client.DownloadStream(ctx, nil)
	if err != nil {
		return fmt.Errorf("download blob: %w", err)
	}
	defer download.Body.Close()

	// Stream the content so memory use does not grow with the blob size.
	hash := sha256.New()
	size, err := io.Copy(hash, download.Body)
	if err != nil {
		return fmt.Errorf("read blob: %w", err)
	}
	slog.InfoContext(ctx, "blob processed",
		"bytes", size,
		"sha256", fmt.Sprintf("%x", hash.Sum(nil)),
	)
	return nil
}
