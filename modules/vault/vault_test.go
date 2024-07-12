package vault_test

import (
	"context"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	vaultClient "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"github.com/tidwall/gjson"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"

	"github.com/testcontainers/testcontainers-go"
	testcontainervault "github.com/testcontainers/testcontainers-go/modules/vault"
)

const (
	token = "root-token"
)

func TestVault(t *testing.T) {
	ctx := context.Background()
	opts := []testcontainers.ContainerCustomizer{
		// WithToken {
		testcontainervault.WithToken(token),
		// }
		// WithInitCommand {
		testcontainervault.WithInitCommand("secrets enable transit", "write -f transit/keys/my-key"),
		testcontainervault.WithInitCommand("kv put secret/test1 foo1=bar1"),
		// }
	}

	vaultContainer, err := testcontainervault.Run(ctx, "hashicorp/vault:1.13.0", opts...)
	assert.NilError(t, err)

	// httpHostAddress {
	hostAddress, err := vaultContainer.HttpHostAddress(ctx)
	// }
	assert.NilError(t, err)

	t.Run("Get secret path", func(t *testing.T) {
		t.Run("From vault CLI", func(t *testing.T) {
			ctx := context.Background()

			// containerCliRead {
			exec, reader, err := vaultContainer.Exec(ctx, []string{"vault", "kv", "get", "-format=json", "secret/test1"})
			// }
			assert.NilError(t, err)
			assert.Check(t, is.Equal(0, exec))

			bytes, err := io.ReadAll(reader)
			assert.NilError(t, err)

			assert.Check(t, is.Equal("bar1", gjson.Get(string(bytes), "data.data.foo1").String()))
		})

		t.Run("From HTTP request", func(t *testing.T) {
			// httpRead {
			request, _ := http.NewRequest(http.MethodGet, hostAddress+"/v1/secret/data/test1", nil)
			request.Header.Add("X-Vault-Token", token)

			response, err := http.DefaultClient.Do(request)
			// }
			assert.NilError(t, err)
			defer response.Body.Close()

			body, err := io.ReadAll(response.Body)
			assert.NilError(t, err)

			assert.Check(t, is.Equal("bar1", gjson.Get(string(body), "data.data.foo1").String()))
		})

		t.Run("From vault client library", func(t *testing.T) {
			ctx := context.Background()

			// clientLibRead {
			client, err := vaultClient.New(
				vaultClient.WithAddress(hostAddress),
				vaultClient.WithRequestTimeout(30*time.Second),
			)
			assert.NilError(t, err)

			err = client.SetToken(token)
			assert.NilError(t, err)

			s, err := client.Secrets.KvV2Read(ctx, "test1", vaultClient.WithMountPath("secret"))
			// }
			assert.NilError(t, err)
			assert.Check(t, is.Equal("bar1", s.Data.Data["foo1"]))
		})
	})

	t.Run("Write secret", func(t *testing.T) {
		t.Run("From vault client library", func(t *testing.T) {
			client, err := vaultClient.New(
				vaultClient.WithAddress(hostAddress),
				vaultClient.WithRequestTimeout(30*time.Second),
			)
			assert.NilError(t, err)

			err = client.SetToken(token)
			assert.NilError(t, err)

			_, err = client.Secrets.KvV2Write(ctx, "test3", schema.KvV2WriteRequest{
				Data: map[string]any{
					"foo": "bar",
				},
			},
				vaultClient.WithMountPath("secret"))
			assert.NilError(t, err)

			s, err := client.Secrets.KvV2Read(ctx, "test3", vaultClient.WithMountPath("secret"))
			assert.NilError(t, err)
			assert.Check(t, is.Equal("bar", s.Data.Data["foo"]))
		})
	})

	t.Cleanup(func() {
		// Clean up the vault after the test is complete
		if err := vaultContainer.Terminate(ctx); err != nil {
			log.Fatalf("failed to terminate vault: %s", err)
		}
	})
}
