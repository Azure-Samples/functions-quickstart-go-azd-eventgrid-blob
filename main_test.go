package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/azure/azure-functions-golang-worker/sdk"
)

func TestProcessBlob(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"empty", http.StatusOK, ""},
		{"large", http.StatusOK, strings.Repeat("sample", 200000)},
		{"missing", http.StatusNotFound, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				if r.Method != http.MethodGet {
					t.Errorf("method = %s, want GET", r.Method)
				}
				w.Header().Set("Content-Length", fmt.Sprint(len(tc.body)))
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client, err := blob.NewClientWithNoCredential(server.URL+"/samples-workitems/test.txt",
				&blob.ClientOptions{ClientOptions: azcore.ClientOptions{
					Retry: policy.RetryOptions{MaxRetries: -1},
				}})
			if err != nil {
				t.Fatal(err)
			}
			err = processBlob(context.Background(), client)
			if (err != nil) != (tc.status != http.StatusOK) {
				t.Fatalf("processBlob error = %v, status = %d", err, tc.status)
			}
			if !called {
				t.Fatal("blob was not downloaded")
			}
		})
	}
}

func TestBlobRegistration(t *testing.T) {
	count := 0
	newFunctionApp().GetRegisteredFunctions().Range(func(_, value any) bool {
		count++
		fn := value.(*sdk.RegisteredFunction)
		binding := fn.TriggerBinding()
		if fn.FuncName != "processBlob" || binding == nil || binding.BlobBinding == nil {
			t.Fatal("missing processBlob trigger")
		}
		b := binding.BlobBinding
		if b.Source != "EventGrid" || b.Path != "samples-workitems/{name}" || b.Connection != "AzureWebJobsStorage" {
			t.Fatalf("unexpected blob configuration: %+v", b)
		}
		return true
	})
	if count != 1 {
		t.Fatalf("registered functions = %d, want 1", count)
	}
}
