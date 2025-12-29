package sql

import (
	"context"
	"fmt"
	"os/exec"
	"time"
)

// ProxyManager manages Cloud SQL Proxy or gcloud proxy processes
type ProxyManager struct {
	cmd             *exec.Cmd
	instanceConnName string
	localPort        int
	usePrivateIP     bool
	useGcloud        bool // if true, use gcloud instead of cloud-sql-proxy
}

// ProxyConfig configures the proxy manager
type ProxyConfig struct {
	InstanceConnectionName string
	LocalPort              int  // Local port to bind (default: 5432)
	UsePrivateIP           bool
	UseGcloud              bool // Use gcloud command instead of cloud-sql-proxy binary
}

// NewProxyManager creates a new proxy manager
func NewProxyManager(config ProxyConfig) *ProxyManager {
	if config.LocalPort == 0 {
		config.LocalPort = 5432
	}
	
	return &ProxyManager{
		instanceConnName: config.InstanceConnectionName,
		localPort:        config.LocalPort,
		usePrivateIP:     config.UsePrivateIP,
		useGcloud:        config.UseGcloud,
	}
}

// Start launches the proxy process in the background
func (pm *ProxyManager) Start(ctx context.Context) error {
	if pm.useGcloud {
		return pm.startGcloudProxy(ctx)
	}
	return pm.startCloudSQLProxy(ctx)
}

// startGcloudProxy starts the proxy using gcloud command
func (pm *ProxyManager) startGcloudProxy(ctx context.Context) error {
	// gcloud sql connect is interactive, we need cloud-sql-proxy or alpha sql proxy
	// Use: gcloud beta sql connect with --tunnel flag OR cloud_sql_proxy
	
	// Extract components from connection name
	project := pm.getProject()
	instance := pm.getInstanceName()
	
	if project == "" || instance == "" {
		return fmt.Errorf("invalid connection name format, expected project:region:instance")
	}
	
	// Use gcloud beta sql proxy (formerly alpha)
	args := []string{
		"beta",
		"sql",
		"connect",
		instance,
		"--project", project,
		"--port", fmt.Sprintf("%d", pm.localPort),
	}
	
	if pm.usePrivateIP {
		args = append(args, "--private-ip")
	}
	
	pm.cmd = exec.CommandContext(ctx, "gcloud", args...)
	
	if err := pm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start gcloud proxy: %w", err)
	}
	
	// Wait longer for the proxy to initialize and be ready
	fmt.Println("Waiting for proxy to be ready...")
	time.Sleep(8 * time.Second)
	
	return nil
}

// startCloudSQLProxy starts the proxy using cloud-sql-proxy binary
func (pm *ProxyManager) startCloudSQLProxy(ctx context.Context) error {
	// cloud-sql-proxy v2 syntax:
	// cloud-sql-proxy --port 5432 PROJECT:REGION:INSTANCE
	// With private IP: add --private-ip flag
	
	args := []string{
		fmt.Sprintf("--port=%d", pm.localPort),
	}
	
	if pm.usePrivateIP {
		args = append(args, "--private-ip")
	}
	
	// Add instance connection name at the end
	args = append(args, pm.instanceConnName)
	
	// Try different possible binary names/paths
	binaryNames := []string{
		"cloud-sql-proxy",
		"cloud_sql_proxy",
		"./cloud-sql-proxy",
		"/nix/store/jrh7phms8710mlmhfpfwjwlg5nawj3mi-google-cloud-sql-proxy-2.19.0/bin/cloud-sql-proxy",
	}
	
	var lastErr error
	for _, binary := range binaryNames {
		pm.cmd = exec.CommandContext(ctx, binary, args...)
		if err := pm.cmd.Start(); err == nil {
			// Wait longer for the proxy to initialize and check logs
			fmt.Printf("Started %s (PID: %d), waiting for it to be ready...\n", binary, pm.cmd.Process.Pid)
			time.Sleep(8 * time.Second)
			
			// Check if process is still running
			if pm.IsRunning() {
				fmt.Println("Proxy process is running and ready")
				return nil
			} else {
				return fmt.Errorf("proxy process exited unexpectedly")
			}
		} else {
			lastErr = err
		}
	}
	
	return fmt.Errorf("failed to start cloud-sql-proxy (tried %v): %w", binaryNames, lastErr)
}

// Stop terminates the proxy process
func (pm *ProxyManager) Stop() error {
	if pm.cmd == nil || pm.cmd.Process == nil {
		return nil
	}
	
	if err := pm.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill proxy process: %w", err)
	}
	
	// Wait for process to exit
	_ = pm.cmd.Wait()
	
	return nil
}

// IsRunning checks if the proxy is running
func (pm *ProxyManager) IsRunning() bool {
	if pm.cmd == nil || pm.cmd.Process == nil {
		return false
	}
	
	// Check if process still exists
	return pm.cmd.ProcessState == nil || !pm.cmd.ProcessState.Exited()
}

// GetLocalPort returns the local port the proxy is listening on
func (pm *ProxyManager) GetLocalPort() int {
	return pm.localPort
}

// getInstanceName extracts instance name from connection string
// Format: project:region:instance -> instance
func (pm *ProxyManager) getInstanceName() string {
	parts := splitConnectionName(pm.instanceConnName)
	if len(parts) == 3 {
		return parts[2]
	}
	return pm.instanceConnName
}

// getProject extracts project from connection string
// Format: project:region:instance -> project
func (pm *ProxyManager) getProject() string {
	parts := splitConnectionName(pm.instanceConnName)
	if len(parts) == 3 {
		return parts[0]
	}
	return ""
}

// splitConnectionName splits project:region:instance format
func splitConnectionName(connName string) []string {
	result := make([]string, 0, 3)
	current := ""
	
	for _, char := range connName {
		if char == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	
	if current != "" {
		result = append(result, current)
	}
	
	return result
}
