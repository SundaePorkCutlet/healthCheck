package healthcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerConfig는 개별 서버 설정을 담는 구조체입니다
type ServerConfig struct {
	IP        string   // 서버 IP 또는 호스트명
	Username  string   // SSH 사용자 이름
	Password  string   // SSH 비밀번호
	Processes []string // 확인할 프로세스 목록
	Type      string   // 서버 유형 (기본 프로세스 목록 선택에 사용)
}

// Config는 헬스 체크 설정을 담는 구조체입니다
type Config struct {
	Servers         []ServerConfig      // 확인할 서버 목록
	SlackWebhookURL string              // Slack 웹훅 URL
	ProcessMap      map[string][]string // 서버 유형별 기본 프로세스 목록
	Commands        map[string]string   // 실행할 명령어 목록
	Thresholds      map[string]float64  // 경고 임계값
}

// NewDefaultConfig는 기본 설정으로 Config를 생성합니다
func NewDefaultConfig() *Config {
	return &Config{
		Servers:         []ServerConfig{},
		SlackWebhookURL: "",
		ProcessMap: map[string][]string{
			"default": {"nginx"},
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

// AddServer는 새 서버를 설정에 추가합니다
func (c *Config) AddServer(ip, username, password string, processes []string, serverType string) {
	c.Servers = append(c.Servers, ServerConfig{
		IP:        ip,
		Username:  username,
		Password:  password,
		Processes: processes,
		Type:      serverType,
	})
}

// AddProcessType은 서버 유형별 프로세스 목록을 추가합니다
func (c *Config) AddProcessType(typeName string, processes []string) {
	c.ProcessMap[typeName] = processes
}

// AddCommand는 새 명령어를 추가하거나 기존 명령어를 수정합니다
func (c *Config) AddCommand(name, command string) {
	c.Commands[name] = command
}

// SetThreshold는 경고 임계값을 설정합니다
func (c *Config) SetThreshold(name string, value float64) {
	c.Thresholds[name] = value
}

// RunCheck는 서버 상태를 확인하고 결과를 반환합니다
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

// getProcessesForServer는 서버에 맞는 프로세스 목록을 반환합니다
func (c *Config) getProcessesForServer(server ServerConfig) []string {
	// 서버에 지정된 프로세스가 있으면 그것을 사용
	if len(server.Processes) > 0 {
		return server.Processes
	}

	// 서버 유형이 지정되어 있고, 해당 유형의 프로세스 목록이 있으면 사용
	if server.Type != "" {
		if processes, ok := c.ProcessMap[server.Type]; ok {
			return processes
		}
	}

	// 기본값 사용
	return c.ProcessMap["default"]
}

// createSSHClient는 SSH 클라이언트를 생성합니다
func createSSHClient(server ServerConfig) (*ssh.Client, error) {
	// 로컬호스트인 경우 SSH 연결 없이 로컬 명령어 실행
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
		return nil, fmt.Errorf("SSH 연결 실패: %v", err)
	}

	return client, nil
}

// runCommand는 명령어를 실행하고 결과를 반환합니다
func runCommand(client *ssh.Client, command string, isLocal bool) (string, error) {
	if isLocal {
		// 로컬 명령어 실행
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}

	// SSH를 통한 원격 명령어 실행
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

// checkStatus는 특정 서버의 상태를 확인합니다
func (c *Config) checkStatus(server ServerConfig) string {
	processes := c.getProcessesForServer(server)
	var result string

	// SSH 클라이언트 생성
	isLocal := server.IP == "localhost" || server.IP == "127.0.0.1"
	client, err := createSSHClient(server)
	if err != nil {
		result += fmt.Sprintf("❌ *SSH Connection:* %s\n", err.Error())
		return result
	}

	// 로컬이 아니고 SSH 연결에 성공한 경우에만 연결 종료
	if !isLocal && client != nil {
		defer client.Close()
	}

	// 시스템 상태 확인 명령어 실행
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

	// 프로세스 실행 상태 확인
	for _, process := range processes {
		cmdStr := fmt.Sprintf("ps aux | grep -v grep | grep '%s'", process)
		output, err := runCommand(client, cmdStr, isLocal)
		if err != nil || output == "" {
			result += fmt.Sprintf("❌ *Process Check:* %s is NOT running\n", process)
		} else {
			result += fmt.Sprintf("✅ *Process Check:* %s is running\n", process)
		}
	}

	return result
}

// sendReportToSlack은 결과를 Slack으로 전송합니다
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
