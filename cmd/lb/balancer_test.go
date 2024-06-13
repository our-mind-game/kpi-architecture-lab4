package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthChecker(t *testing.T) {
	hc := &HealthChecker{
		serverHealthStatus: map[string]bool{},
		health:             mockHealth,
	}

	hc.CheckAllServers()
	assert.Equal(t, map[string]bool{"server1:8080": true, "server2:8080": false, "server3:8080": false}, hc.serverHealthStatus)

	healthyServers := hc.GetHealthyServers()
	assert.Equal(t, []string{"server1:8080"}, healthyServers)

	hc.health = mockHealthTrue
	hc.CheckAllServers()
	healthyServers = hc.GetHealthyServers()
	assert.Equal(t, []string{"server1:8080", "server2:8080", "server3:8080"}, healthyServers)
}

func TestBalancer(t *testing.T) {
	hc := &HealthChecker{
		serverHealthStatus: map[string]bool{
			"server1:8080": true,
			"server2:8080": true,
			"server3:8080": true,
		},
	}

	lb := &LoadBalancer{
		healthChecker: hc,
	}

	server1 := lb.GetAppropriateServer("/check")
	server1SecondReq := lb.GetAppropriateServer("/check")
	server2 := lb.GetAppropriateServer("/check2")
	server3 := lb.GetAppropriateServer("/check5")

	assert.Equal(t, "server1:8080", server1)
	assert.Equal(t, server1, server1SecondReq)
	assert.Equal(t, "server2:8080", server2)
	assert.Equal(t, "server3:8080", server3)
}

func TestBalancer_AllServersUnavailable(t *testing.T) {
	hc := &HealthChecker{
		serverHealthStatus: map[string]bool{
			"server1:8080": false,
			"server2:8080": false,
			"server3:8080": false,
		},
	}

	lb := &LoadBalancer{
		healthChecker: hc,
	}

	server := lb.GetAppropriateServer("/check")

	assert.Equal(t, "", server)
}

func mockHealth(dst string) bool {
	return dst == "server1:8080"
}

func mockHealthTrue(dst string) bool {
	return true
}
