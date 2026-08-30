package security

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// PortStatus represents a scanned service port
type PortStatus struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
	IsOpen  bool   `json:"is_open"`
	Banner  string `json:"banner,omitempty"`
}

// Standard developer & infrastructure service ports
var standardPorts = map[int]string{
	21:    "FTP",
	22:    "SSH",
	25:    "SMTP",
	53:    "DNS",
	80:    "HTTP",
	443:   "HTTPS",
	3000:  "Node.js / Dev Server",
	3306:  "MySQL",
	5000:  "Flask / Python App",
	5432:  "PostgreSQL",
	6379:  "Redis",
	8000:  "Django / Dev Server",
	8080:  "HTTP Alternate / Spring",
	8443:  "HTTPS Alternate",
	9000:  "SonarQube / Portainer",
	11434: "Ollama Local LLM",
	27017: "MongoDB",
}

// AuditPortConnectivity checks if designated common service ports are accessible
func AuditPortConnectivity(host string, customPorts []int) []PortStatus {
	// Clean host
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.Index(host, "/"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	portsToScan := customPorts
	if len(portsToScan) == 0 {
		for p := range standardPorts {
			portsToScan = append(portsToScan, p)
		}
	}

	var results []PortStatus
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range portsToScan {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			addr := fmt.Sprintf("%s:%d", host, p)
			conn, err := net.DialTimeout("tcp", addr, 1500*time.Millisecond)

			serviceName := standardPorts[p]
			if serviceName == "" {
				serviceName = "Custom Service"
			}

			status := PortStatus{
				Port:    p,
				Service: serviceName,
				IsOpen:  err == nil,
			}

			if err == nil {
				defer conn.Close()
				_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				buf := make([]byte, 256)
				n, _ := conn.Read(buf)
				if n > 0 {
					banner := strings.TrimSpace(string(buf[:n]))
					if len(banner) > 50 {
						banner = banner[:50] + "..."
					}
					status.Banner = banner
				}
			}

			mu.Lock()
			results = append(results, status)
			mu.Unlock()
		}(port)
	}

	wg.Wait()
	return results
}
