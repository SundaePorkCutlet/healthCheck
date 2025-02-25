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

	// 서버 추가 (IP, 사용자 이름, 비밀번호, 프로세스 목록)
	config.AddServer("localhost", "user1", "password1", []string{"nginx", "prometheus"})
	config.AddServer("10.1.0.170", "user2", "password2", []string{"fluentd", "nginx"})
	config.AddServer("10.1.0.172", "user3", "password3", []string{}) // 빈 프로세스 목록은 기본값 사용

	// 헬스 체크 실행
	report := config.RunCheck()
	fmt.Println("Health check completed!")
	fmt.Println(report)
}
