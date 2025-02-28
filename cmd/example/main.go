package main

import (
	"fmt"

	healthcheck "github.com/SundaePorkCutlet/healthCheck"
)

func main() {
	// 기본 설정으로 Config 생성
	config := healthcheck.NewDefaultConfig()

	// Slack Webhook URL 설정
	config.SlackWebhookURL = "https://hooks.slack.com/services/YOUR_WEBHOOK_URL"

	// 서버 유형별 프로세스 목록 설정
	config.AddProcessType("web", []string{"nginx", "prometheus"})
	config.AddProcessType("app", []string{"api-server", "auth-service"})
	config.AddProcessType("db", []string{"mysql", "redis-server"})

	// 서버 추가 (IP, 사용자 이름, 비밀번호, 프로세스 목록, 서버 유형)
	config.AddServer("10.1.0.170", "user", "password", []string{}, "web") // 웹 서버
	config.AddServer("10.1.0.171", "user", "password", []string{}, "app") // 애플리케이션 서버
	config.AddServer("10.1.0.172", "user", "password",
		[]string{"custom-service"}, // 추가 프로세스
		"db")                       // 데이터베이스 서버

	// 헬스 체크 실행
	report := config.RunCheck()
	fmt.Println("Health check completed!")
	fmt.Println(report)
}
