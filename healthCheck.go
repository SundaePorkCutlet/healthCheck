package healthcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ServerConfig는 개별 서버 설정을 담는 구조체입니다
type ServerConfig struct {
	IP        string
	Username  string
	Password  string
	Processes []string
}

// Config는 헬스 체크 설정을 담는 구조체입니다
type Config struct {
	Servers           []ServerConfig
	SlackWebhookURL   string
	DefaultProcessMap map[string][]string
	Commands          map[string]string
	Thresholds        map[string]float64
}

// NewDefaultConfig는 기본 설정으로 Config를 생성합니다
func NewDefaultConfig() *Config {
	return &Config{
		Servers:         []ServerConfig{},
		SlackWebhookURL: "",
		DefaultProcessMap: map[string][]string{
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
func (c *Config) AddServer(ip, username, password string, processes []string) {
	c.Servers = append(c.Servers, ServerConfig{
		IP:        ip,
		Username:  username,
		Password:  password,
		Processes: processes,
	})
}

// AddProcessType은 서버 유형별 프로세스 목록을 추가합니다
func (c *Config) AddProcessType(typeName string, processes []string) {
	c.DefaultProcessMap[typeName] = processes
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

	// 서버 IP에 따라 기본값 사용
	if server.IP == "localhost" && len(c.DefaultProcessMap["localhost"]) > 0 {
		return c.DefaultProcessMap["localhost"]
	} else if strings.HasPrefix(server.IP, "10.1.0.17") && len(c.DefaultProcessMap["server"]) > 0 {
		return c.DefaultProcessMap["server"]
	} else if len(c.DefaultProcessMap["client"]) > 0 {
		return c.DefaultProcessMap["client"]
	}

	// 해당 유형이 없으면 기본값 사용
	return c.DefaultProcessMap["default"]
}

// checkStatus는 특정 서버의 상태를 확인합니다
func (c *Config) checkStatus(server ServerConfig) string {
	processes := c.getProcessesForServer(server)
	var result string

	// 시스템 상태 확인 명령어 실행
	for label, cmdStr := range c.Commands {
		output, err := exec.Command("sshpass", "-p", server.Password, "ssh", fmt.Sprintf("%s@%s", server.Username, server.IP), cmdStr).CombinedOutput()
		outputStr := strings.TrimSpace(string(output))

		if err != nil {
			result += fmt.Sprintf("❌ *%s:* Error executing command on %s - %s\n", label, server.IP, err.Error())
			continue
		}

		switch label {
		case "CPU Usage":
			idleStr := strings.Split(outputStr, ", ")[2] // "xx% idle"
			idlePercent, _ := strconv.ParseFloat(strings.TrimSuffix(strings.Fields(idleStr)[0], "%"), 64)
			if idlePercent <= c.Thresholds["CPU Idle"] {
				result += fmt.Sprintf("⚠️ *%s:* %s\n", label, outputStr)
			} else {
				result += fmt.Sprintf("✅ *%s:* %s\n", label, outputStr)
			}

		case "Memory Usage":
			fields := strings.Fields(outputStr) // ["xxGi" "total," "xxGi" "used," "xxGi" "free"]
			usedStr := strings.TrimSuffix(fields[2], "Gi")
			totalStr := strings.TrimSuffix(fields[0], "Gi")
			used, _ := strconv.ParseFloat(usedStr, 64)
			total, _ := strconv.ParseFloat(totalStr, 64)
			usagePercent := (used / total) * 100
			if usagePercent >= c.Thresholds["Memory Used"] {
				result += fmt.Sprintf("⚠️ *%s:* %s (%.1f%% used)\n", label, outputStr, usagePercent)
			} else {
				result += fmt.Sprintf("✅ *%s:* %s (%.1f%% used)\n", label, outputStr, usagePercent)
			}

		case "Disk Usage":
			usedStr := strings.Split(outputStr, ", ")[2] // "xx% used"
			usedPercent, _ := strconv.ParseFloat(strings.TrimSuffix(strings.Fields(usedStr)[0], "%"), 64)
			if usedPercent >= c.Thresholds["Disk Used"] {
				result += fmt.Sprintf("⚠️ *%s:* %s\n", label, outputStr)
			} else {
				result += fmt.Sprintf("✅ *%s:* %s\n", label, outputStr)
			}

		default:
			result += fmt.Sprintf("✅ *%s:* %s\n", label, outputStr)
		}
	}

	// 프로세스 실행 상태 확인
	for _, process := range processes {
		cmdStr := fmt.Sprintf("ps aux | grep -v grep | grep '%s'", process)
		output, err := exec.Command("sshpass", "-p", server.Password, "ssh", fmt.Sprintf("%s@%s", server.Username, server.IP), cmdStr).CombinedOutput()
		outputStr := strings.TrimSpace(string(output))

		if err != nil || outputStr == "" {
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

	cmd := exec.Command("curl", "-X", "POST", c.SlackWebhookURL, "-H", "Content-Type: application/json", "-d", string(jsonPayload))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error sending report to Slack: %v\n", err)
	} else {
		fmt.Println("Report sent to Slack successfully!")
	}
}
