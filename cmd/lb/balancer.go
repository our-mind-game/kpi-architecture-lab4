package main

import (
	"context"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/our-mind-game/kpi-architecture-lab4/httptools"
	"github.com/our-mind-game/kpi-architecture-lab4/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

type LoadBalancer struct {
	healthChecker *HealthChecker
}

func (lb *LoadBalancer) CheckServers() {
	for {
		lb.healthChecker.CheckAllServers()
		time.Sleep(10 * time.Second)
	}
}

func (lb *LoadBalancer) GetServerIndex(reqUrl string) int {
	h := fnv.New32a()
	h.Write([]byte(reqUrl))
	healthyServers := lb.healthChecker.GetHealthyServers()
	return int(h.Sum32() % uint32(len(healthyServers)))
}

func (lb *LoadBalancer) GetAppropriateServer(urlPath string) string {
	healthyServers := lb.healthChecker.GetHealthyServers()

	if len(healthyServers) == 0 {
		log.Println("No servers available")
		return ""
	}

	return healthyServers[lb.GetServerIndex(urlPath)]
}

type HealthChecker struct {
	serverHealthStatus map[string]bool
	health             func(dst string) bool
}

func (hc *HealthChecker) CheckAllServers() {
	for _, server := range serversPool {
		hc.serverHealthStatus[server] = hc.health(server)
	}
}

func (hc *HealthChecker) GetHealthyServers() []string {
	var healthyServers []string
	for _, server := range serversPool {
		if hc.serverHealthStatus[server] {
			healthyServers = append(healthyServers, server)
		}
	}
	return healthyServers
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func main() {
	flag.Parse()

	healthChecker := &HealthChecker{
		serverHealthStatus: map[string]bool{},
		health:             health,
	}

	balancer := &LoadBalancer{
		healthChecker: healthChecker,
	}

	go balancer.CheckServers()

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		server := balancer.GetAppropriateServer(r.URL.Path)
		forward(server, rw, r)
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
