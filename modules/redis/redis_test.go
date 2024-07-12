package redis_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"gotest.tools/v3/assert"

	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func TestIntegrationSetGet(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := tcredis.Run(ctx, "docker.io/redis:7")
	assert.NilError(t, err)
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	assertSetsGets(t, ctx, redisContainer, 1)
}

func TestRedisWithConfigFile(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := tcredis.Run(ctx, "docker.io/redis:7", tcredis.WithConfigFile(filepath.Join("testdata", "redis7.conf")))
	assert.NilError(t, err)
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	assertSetsGets(t, ctx, redisContainer, 1)
}

func TestRedisWithImage(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		image string
	}{
		{
			name:  "Redis6",
			image: "docker.io/redis:6",
		},
		{
			name:  "Redis7",
			image: "docker.io/redis:7",
		},
		{
			name: "Redis Stack",
			// redisStackImage {
			image: "docker.io/redis/redis-stack:latest",
			// }
		},
		{
			name: "Redis Stack Server",
			// redisStackServerImage {
			image: "docker.io/redis/redis-stack-server:latest",
			// }
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisContainer, err := tcredis.Run(ctx, tt.image, tcredis.WithConfigFile(filepath.Join("testdata", "redis6.conf")))
			assert.NilError(t, err)
			t.Cleanup(func() {
				if err := redisContainer.Terminate(ctx); err != nil {
					t.Fatalf("failed to terminate container: %s", err)
				}
			})

			assertSetsGets(t, ctx, redisContainer, 1)
		})
	}
}

func TestRedisWithLogLevel(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := tcredis.Run(ctx, "docker.io/redis:7", tcredis.WithLogLevel(tcredis.LogLevelVerbose))
	assert.NilError(t, err)
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	assertSetsGets(t, ctx, redisContainer, 10)
}

func TestRedisWithSnapshotting(t *testing.T) {
	ctx := context.Background()

	redisContainer, err := tcredis.Run(ctx, "docker.io/redis:7", tcredis.WithSnapshotting(10, 1))
	assert.NilError(t, err)
	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	assertSetsGets(t, ctx, redisContainer, 10)
}

func assertSetsGets(t *testing.T, ctx context.Context, redisContainer *tcredis.RedisContainer, keyCount int) {
	// connectionString {
	uri, err := redisContainer.ConnectionString(ctx)
	// }
	assert.NilError(t, err)

	// You will likely want to wrap your Redis package of choice in an
	// interface to aid in unit testing and limit lock-in throughout your
	// codebase but that's out of scope for this example
	options, err := redis.ParseURL(uri)
	assert.NilError(t, err)

	client := redis.NewClient(options)
	defer func(t *testing.T, ctx context.Context, client *redis.Client) {
		assert.NilError(t, flushRedis(ctx, *client))
	}(t, ctx, client)

	t.Log("pinging redis")
	pong, err := client.Ping(ctx).Result()
	assert.NilError(t, err)

	t.Log("received response from redis")

	if pong != "PONG" {
		t.Fatalf("received unexpected response from redis: %s", pong)
	}

	for i := 0; i < keyCount; i++ {
		// Set data
		key := fmt.Sprintf("{user.%s}.favoritefood.%d", uuid.NewString(), i)
		value := fmt.Sprintf("Cabbage Biscuits %d", i)

		ttl, _ := time.ParseDuration("2h")
		err = client.Set(ctx, key, value, ttl).Err()
		assert.NilError(t, err)

		// Get data
		savedValue, err := client.Get(ctx, key).Result()
		assert.NilError(t, err)

		if savedValue != value {
			t.Fatalf("Expected value %s. Got %s.", savedValue, value)
		}
	}
}

func flushRedis(ctx context.Context, client redis.Client) error {
	return client.FlushAll(ctx).Err()
}
