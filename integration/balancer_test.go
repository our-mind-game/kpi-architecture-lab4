package integration

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	response1, err := client.Get(fmt.Sprintf("%s/api/v1/some-data2", baseAddress))
	require.NoError(t, err)
	response1Header := response1.Header.Get("lb-from")
	assert.Equal(t, "server1:8080", response1Header)

	response2, err := client.Get(fmt.Sprintf("%s/api/v1/some-data5", baseAddress))
	require.NoError(t, err)
	response2Header := response2.Header.Get("lb-from")
	assert.Equal(t, "server2:8080", response2Header)

	response3, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
	require.NoError(t, err)
	response3Header := response3.Header.Get("lb-from")
	assert.Equal(t, "server3:8080", response3Header)

	response1Repeat, err := client.Get(fmt.Sprintf("%s/api/v1/some-data2", baseAddress))
	require.NoError(t, err)
	response1RepeatHeader := response1Repeat.Header.Get("lb-from")
	assert.Equal(t, "server1:8080", response1RepeatHeader)
}

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	for i := 0; i < b.N; i++ {
		_, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		require.NoError(b, err)
	}
}
