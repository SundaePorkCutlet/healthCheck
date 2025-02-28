# Server Health Check

[English](#english) | [한국어](#korean)

# English

A Go library for checking server status and reporting to Slack.

## Requirements

- Linux operating system (required)
- sshpass command (for server access)
- curl command (for Slack message delivery)

## Installation
```bash
go get github.com/SundaePorkCutlet/healthCheck
```


## Basic Usage
```go
package main
import (
"fmt"
"github.com/SundaePorkCutlet/healthCheck/healthcheck"
)
func main() {
// Create config with default settings
config := healthcheck.NewDefaultConfig()
// Set Slack Webhook URL
config.SlackWebhookURL = "https://hooks.slack.com/services/YOUR_WEBHOOK_URL"
// Add servers to monitor
config.AddServer("10.1.0.170", "user", "password", []string{"nginx", "prometheus"})
// Run health check
report := config.RunCheck()
fmt.Println("Health check completed!")
}
```


## Advanced Features

### Process Type Management

```go
// Set default processes for different server types
config.AddProcessType("web", []string{"nginx", "prometheus"})
config.AddProcessType("app", []string{"api-server", "auth-service"})
```

### Custom Commands and Thresholds
```go
// Add custom commands
config.AddCommand("Docker Status", "systemctl status docker | grep Active")
// Set custom thresholds
config.SetThreshold("CPU Idle", 15.0) // Warning if CPU idle is below 15%
config.SetThreshold("Memory Used", 90.0) // Warning if memory usage exceeds 90%
config.SetThreshold("Disk Used", 95.0) // Warning if disk usage exceeds 95%
```


## Features

- Customizable process monitoring per server
- Default process lists by server type
- Custom command execution
- Adjustable warning thresholds
- System metrics monitoring:
  - CPU usage
  - Memory usage
  - Disk usage
  - Network connectivity
  - Process status
- Slack integration for reporting

---

# Korean

서버 상태를 확인하고 Slack으로 보고하는 Go 라이브러리입니다.

## 요구사항

- Linux 운영체제 (필수)
- sshpass 명령어 (서버 접속용)
- curl 명령어 (Slack 메시지 전송용)

## 설치

```bash
go get github.com/SundaePorkCutlet/healthCheck
```

## 기본 사용법
```go
package main
import (
"fmt"
"github.com/SundaePorkCutlet/healthCheck/healthcheck"
)
func main() {
// 기본 설정으로 Config 생성
config := healthcheck.NewDefaultConfig()
// Slack Webhook URL 설정
config.SlackWebhookURL = "https://hooks.slack.com/services/YOUR_WEBHOOK_URL"
// 서버 추가
config.AddServer("10.1.0.170", "user", "password", []string{"nginx", "prometheus"})
// 헬스 체크 실행
report := config.RunCheck()
fmt.Println("Health check completed!")
}
```

## 고급 기능

### 프로세스 타입 관리

```go
// 서버 유형별 기본 프로세스 목록 설정
config.AddProcessType("web", []string{"nginx", "prometheus"})
config.AddProcessType("app", []string{"api-server", "auth-service"})
```

### 사용자 정의 명령어 및 임계값

```go
// 사용자 정의 명령어 추가
config.AddCommand("Docker Status", "systemctl status docker | grep Active")
// 임계값 설정
config.SetThreshold("CPU Idle", 15.0) // CPU 사용률이 15% 이하일 때 경고
config.SetThreshold("Memory Used", 90.0) // 메모리 사용률이 90% 이상일 때 경고
config.SetThreshold("Disk Used", 95.0) // 디스크 사용률이 95% 이상일 때 경고
```

## 기능

- 서버별 프로세스 모니터링 커스터마이징
- 서버 유형별 기본 프로세스 목록 설정
- 커스텀 명령어 실행
- 경고 임계값 조정
- 시스템 메트릭 모니터링:
  - CPU 사용량
  - 메모리 사용량
  - 디스크 사용량
  - 네트워크 연결 상태
  - 프로세스 상태
- Slack 연동 보고

## License

MIT
