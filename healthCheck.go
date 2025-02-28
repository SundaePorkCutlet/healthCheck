package healthcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerConfig holds configuration for individual server
type ServerConfig struct {
	IP        string   // Server IP or hostname
	Username  string   // SSH username
	Password  string   // SSH password
	Processes []string // List of processes to monitor
	Type      string   // Server type (used for default process selection)
}

// Config holds health check configuration
type Config struct {
	Servers         []ServerConfig      // List of servers to monitor
	SlackWebhookURL string              // Slack webhook URL
	ProcessMap      map[string][]string // Default process list by server type
	Commands        map[string]string   // List of commands to execute
	Thresholds      map[string]float64  // Warning thresholds
}

// NewDefaultConfig creates a Config with default settings
func NewDefaultConfig() *Config {
	// Linux system check
	if runtime.GOOS != "linux" {
		panic("This program is only supported on Linux operating systems.")
	}

	return &Config{
		Servers:         []ServerConfig{},
		SlackWebhookURL: "",
		ProcessMap: map[string][]string{
			"default": {}, // Empty default value
		},
		Commands: map[string]string{
			"CPU Usage":     "top -bn1 | grep 'Cpu(s)' | awk '{print $2 \"% user, \" $4 \"% system, \" $8 \"% idle\"}'",
			"Memory Usage":  "free -h | awk 'NR==2{print $2 \" total, \" $3 \" used, \" $4 \" free\"}'",
			"Disk Usage":    "df -h | awk '$NF==\"/\"{print $2 \" total, \" $3 \" used, \" $5 \" used\"}'",
			"Network Check": "ping -c 1 8.8.8.8 > /dev/null && echo 'Network is OK' || echo 'Network Issue'",
		},
		Thresholds: map[string]float64{
			"CPU Idle":    20.0,
			"Memory Used": 80.0,
			"Disk Used":   90.0,
		},
	}
}

// AddServer adds a new server to the configuration
func (c *Config) AddServer(ip, username, password string, processes []string, serverType string) {
	c.Servers = append(c.Servers, ServerConfig{
		IP:        ip,
		Username:  username,
		Password:  password,
		Processes: processes,
		Type:      serverType,
	})
}

// AddProcessType adds a process list for a server type
func (c *Config) AddProcessType(typeName string, processes []string) {
	c.ProcessMap[typeName] = processes
}

// AddCommand adds a new command or modifies an existing one
func (c *Config) AddCommand(name, command string) {
	c.Commands[name] = command
}

// SetThreshold sets a warning threshold
func (c *Config) SetThreshold(name string, value float64) {
	c.Thresholds[name] = value
}

// RunCheck checks the status of all servers and returns the result
func (c *Config) RunCheck() string {
	var report string

	for _, server := range c.Servers {
		fmt.Printf("\n=== Checking status of server: %s ===\n", server.IP)
		report += fmt.Sprintf("\n=== Checking status of server: %s ===\n", server.IP)
		report += c.checkStatus(server)
	}

	if c.SlackWebhookURL != "" {
		c.sendReportToSlack(report)
	}

	return report
}

// getProcessesForServer returns the appropriate process list for a server
func (c *Config) getProcessesForServer(server ServerConfig) []string {
	// If a specific process list is specified for the server, use it
	if len(server.Processes) > 0 {
		return server.Processes
	}

	// If a server type is specified and there is a process list for that type, use it
	if server.Type != "" {
		if processes, ok := c.ProcessMap[server.Type]; ok {
			return processes
		}
	}

	// Use default value
	return c.ProcessMap["default"]
}

// createSSHClient creates an SSH client
func createSSHClient(server ServerConfig) (*ssh.Client, error) {
	// For localhost, execute commands locally without SSH
	if server.IP == "localhost" || server.IP == "127.0.0.1" {
		return nil, nil
	}

	config := &ssh.ClientConfig{
		User: server.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(server.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", server.IP), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect via SSH: %v", err)
	}

	return client, nil
}

// runCommand executes a command and returns the result
func runCommand(client *ssh.Client, command string, isLocal bool) (string, error) {
	if isLocal {
		// Execute local command
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}

	// Execute remote command via SSH
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout
	err = session.Run(command)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// checkStatus checks the status of a specific server
func (c *Config) checkStatus(server ServerConfig) string {
	processes := c.getProcessesForServer(server)
	var result string

	// Create SSH client
	isLocal := server.IP == "localhost" || server.IP == "127.0.0.1"
	client, err := createSSHClient(server)
	if err != nil {
		result += fmt.Sprintf("❌ *SSH Connection:* %s\n", err.Error())
		return result
	}

	if !isLocal && client != nil {
		defer client.Close()
	}

	// Execute system status check commands
	for label, cmdStr := range c.Commands {
		output, err := runCommand(client, cmdStr, isLocal)
		if err != nil {
			result += fmt.Sprintf("❌ *%s:* Error executing command on %s - %s\n", label, server.IP, err.Error())
			continue
		}

		switch label {
		case "CPU Usage":
			idleStr := strings.Split(output, ", ")[2] // "xx% idle"
			idlePercent, _ := strconv.ParseFloat(strings.TrimSuffix(strings.Fields(idleStr)[0], "%"), 64)
			if idlePercent <= c.Thresholds["CPU Idle"] {
				result += fmt.Sprintf("⚠️ *%s:* %s\n", label, output)
			} else {
				result += fmt.Sprintf("✅ *%s:* %s\n", label, output)
			}

		case "Memory Usage":
			fields := strings.Fields(output) // ["xxGi" "total," "xxGi" "used," "xxGi" "free"]
			usedStr := strings.TrimSuffix(fields[2], "Gi")
			totalStr := strings.TrimSuffix(fields[0], "Gi")
			used, _ := strconv.ParseFloat(usedStr, 64)
			total, _ := strconv.ParseFloat(totalStr, 64)
			usagePercent := (used / total) * 100
			if usagePercent >= c.Thresholds["Memory Used"] {
				result += fmt.Sprintf("⚠️ *%s:* %s (%.1f%% used)\n", label, output, usagePercent)
			} else {
				result += fmt.Sprintf("✅ *%s:* %s (%.1f%% used)\n", label, output, usagePercent)
			}

		case "Disk Usage":
			usedStr := strings.Split(output, ", ")[2] // "xx% used"
			usedPercent, _ := strconv.ParseFloat(strings.TrimSuffix(strings.Fields(usedStr)[0], "%"), 64)
			if usedPercent >= c.Thresholds["Disk Used"] {
				result += fmt.Sprintf("⚠️ *%s:* %s\n", label, output)
			} else {
				result += fmt.Sprintf("✅ *%s:* %s\n", label, output)
			}

		default:
			result += fmt.Sprintf("✅ *%s:* %s\n", label, output)
		}
	}

	// Check process status
	if len(processes) > 0 {
		for _, process := range processes {
			cmdStr := fmt.Sprintf("ps aux | grep -v grep | grep '%s'", process)
			output, err := runCommand(client, cmdStr, isLocal)
			if err != nil || output == "" {
				result += fmt.Sprintf("❌ *Process Check:* %s is NOT running\n", process)
			} else {
				result += fmt.Sprintf("✅ *Process Check:* %s is running\n", process)
			}
		}
	} else {
		result += "ℹ️ *Process Check:* No processes specified for monitoring\n"
	}

	return result
}

// sendReportToSlack sends the report to Slack
func (c *Config) sendReportToSlack(report string) {
	currentTime := time.Now().Format("2006-01-02 15:04:05")
	title := fmt.Sprintf("Daily Health Check Report - %s", currentTime)

	payload := map[string]string{
		"text": fmt.Sprintf("*%s*\n%s", title, report),
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return
	}

	resp, err := http.Post(c.SlackWebhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Printf("Error sending report to Slack: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Slack API error: %s - %s\n", resp.Status, string(body))
		return
	}

	fmt.Println("Report sent to Slack successfully!")
}
