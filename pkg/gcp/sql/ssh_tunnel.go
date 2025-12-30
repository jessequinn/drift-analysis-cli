package sql

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"
)

// SSHTunnelManager manages SSH tunnel connections through bastion hosts
type SSHTunnelManager struct {
	config      *SSHTunnelConfig
	cmd         *exec.Cmd
	isConnected bool
}

// getFreePort finds an available port on localhost
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

// NewSSHTunnelManager creates a new SSH tunnel manager
func NewSSHTunnelManager(config *SSHTunnelConfig) (*SSHTunnelManager, error) {
	if config == nil {
		return nil, fmt.Errorf("SSH tunnel config is nil")
	}
	
	// Set defaults
	if config.LocalPort == 0 {
		// Automatically find a free port
		port, err := getFreePort()
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
		config.LocalPort = port
	}
	if config.RemotePort == 0 {
		config.RemotePort = 5432
	}
	if config.SSHKeyExpiry == "" {
		config.SSHKeyExpiry = "1h"
	}
	
	return &SSHTunnelManager{
		config:      config,
		isConnected: false,
	}, nil
}

// Start establishes the SSH tunnel through the bastion host
func (stm *SSHTunnelManager) Start(ctx context.Context) error {
	if stm.isConnected {
		return nil // Already connected
	}

	fmt.Printf("Establishing SSH tunnel via bastion host %s...\n", stm.config.BastionHost)

	// Build gcloud compute ssh command
	args := []string{
		"compute",
		"ssh",
		"--zone", stm.config.BastionZone,
		stm.config.BastionHost,
		"--project", stm.config.Project,
		"--ssh-key-expire-after", stm.config.SSHKeyExpiry,
	}

	// Add IAP tunnel flag if enabled
	if stm.config.UseIAP {
		args = append(args, "--tunnel-through-iap")
	}

	// Add SSH port forwarding
	sshForward := fmt.Sprintf("-N -L localhost:%d:%s:%d",
		stm.config.LocalPort,
		stm.config.PrivateIP,
		stm.config.RemotePort,
	)
	args = append(args, "--ssh-flag="+sshForward)

	// Create command
	stm.cmd = exec.CommandContext(ctx, "gcloud", args...)

	// Start SSH tunnel
	if err := stm.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start SSH tunnel: %w", err)
	}

	fmt.Printf("SSH tunnel started (PID: %d), waiting for it to be ready...\n", stm.cmd.Process.Pid)

	// Wait for tunnel to be ready
	if err := stm.waitForTunnel(30 * time.Second); err != nil {
		stm.Stop()
		return fmt.Errorf("SSH tunnel failed to become ready: %w", err)
	}

	stm.isConnected = true
	fmt.Printf("SSH tunnel established: localhost:%d -> %s:%d\n",
		stm.config.LocalPort,
		stm.config.PrivateIP,
		stm.config.RemotePort,
	)

	return nil
}

// Stop closes the SSH tunnel
func (stm *SSHTunnelManager) Stop() error {
	if !stm.isConnected {
		return nil
	}

	fmt.Println("Closing SSH tunnel...")

	if stm.cmd != nil && stm.cmd.Process != nil {
		if err := stm.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill SSH tunnel process: %w", err)
		}
		// Wait for process to exit
		_ = stm.cmd.Wait()
	}

	stm.isConnected = false
	return nil
}

// IsConnected returns whether the tunnel is currently active
func (stm *SSHTunnelManager) IsConnected() bool {
	if !stm.isConnected {
		return false
	}

	// Check if process is still running
	if stm.cmd == nil || stm.cmd.Process == nil {
		stm.isConnected = false
		return false
	}

	if stm.cmd.ProcessState != nil && stm.cmd.ProcessState.Exited() {
		stm.isConnected = false
		return false
	}

	return true
}

// GetLocalPort returns the local port the tunnel is listening on
func (stm *SSHTunnelManager) GetLocalPort() int {
	return stm.config.LocalPort
}

// waitForTunnel waits for the SSH tunnel to be ready by checking if the local port is listening
func (stm *SSHTunnelManager) waitForTunnel(maxWait time.Duration) error {
	deadline := time.Now().Add(maxWait)
	
	for time.Now().Before(deadline) {
		// Try to connect to the local port
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", stm.config.LocalPort), time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		
		// Check if process is still running
		if stm.cmd.ProcessState != nil && stm.cmd.ProcessState.Exited() {
			return fmt.Errorf("SSH tunnel process exited unexpectedly")
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	return fmt.Errorf("SSH tunnel did not become ready within %v", maxWait)
}

// GetConnectionString returns a connection string that uses the SSH tunnel
func (stm *SSHTunnelManager) GetConnectionString(user, password, database string) string {
	return fmt.Sprintf("host=localhost port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=60",
		stm.config.LocalPort, user, password, database)
}
