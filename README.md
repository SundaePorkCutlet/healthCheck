# Server Health Check

서버 상태를 확인하고 Slack으로 보고하는 Go 라이브러리입니다.

## 설치

go get github.com/SundaePorkCutlet/healthCheck

## 사용법

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
	
	// 서버 유형별 프로세스 목록 설정
	config.AddProcessType("localhost", []string{"fluentd", "prometheus", "nginx"})
	config.AddProcessType("server", []string{"fluentd", "prometheus", "nginx", "hero-auth", "hero-apiserver"})
	config.AddProcessType("client", []string{"fluent-bit", "hero-transcoder-manager"})
	
	// 서버 추가 (IP, 사용자 이름, 비밀번호, 프로세스 목록)
	config.AddServer("localhost", "user1", "password1", []string{"nginx", "prometheus"})
	config.AddServer("10.1.0.170", "user2", "password2", []string{}) // 빈 프로세스 목록은 기본값 사용
	config.AddServer("10.1.0.172", "user3", "password3", []string{}) // 빈 프로세스 목록은 기본값 사용
	
	// 헬스 체크 실행
	report := config.RunCheck()
	fmt.Println("Health check completed!")
}

## 고급 사용법

// 명령어 커스터마이징
config.AddCommand("Docker Status", "systemctl status docker | grep Active")

// 임계값 커스터마이징
config.SetThreshold("CPU Idle", 15.0)  // CPU 유휴 상태가 15% 이하면 경고
config.SetThreshold("Memory Used", 90.0)  // 메모리 사용량이 90% 이상이면 경고
config.SetThreshold("Disk Used", 95.0)  // 디스크 사용량이 95% 이상이면 경고

## 기능

- 각 서버마다 다른 사용자 이름과 비밀번호 지정 가능
- 서버별 모니터링할 프로세스 목록 커스터마이징
- 서버 유형별 기본 프로세스 목록 설정
- 실행할 명령어 커스터마이징
- 경고 임계값 조정
- CPU, 메모리, 디스크 사용량 확인
- 네트워크 연결 확인
- 프로세스 실행 상태 확인
- Slack으로 결과 보고

## 요구사항

- sshpass 명령어 (서버 접속용)
- curl 명령어 (Slack 메시지 전송용)

## 라이센스

MIT