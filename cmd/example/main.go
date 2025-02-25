package main

import (
	"fmt"

	healthcheck "github.com/SundaePorkCutlet/healthCheck"
)

func main() {
	// 기본 설정으로 Config 생성
	config := healthcheck.NewDefaultConfig()

	// Slack Webhook URL 설정
	config.SlackWebhookURL = "https://hooks.slack.com/services/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"

	// 서버 유형별 프로세스 목록 설정
	config.AddProcessType("localhost", []string{"fluentd", "prometheus", "hero-auth", "hero-apiserver", "nginx", "hero-monitor-dashboard-web", "fluent-bit", "hero-transcoder-manager"})
	config.AddProcessType("server", []string{"fluentd", "prometheus", "hero-auth", "hero-apiserver", "nginx"})
	config.AddProcessType("client", []string{"fluent-bit", "hero-transcoder-manager"})

	// 서버 추가 (IP, 사용자 이름, 비밀번호, 프로세스 목록, 서버 유형)
	config.AddServer("localhost", "hera", "mediaExcel(0)", []string{}, "localhost")
	config.AddServer("10.1.0.170", "hera", "mediaExcel(0)", []string{}, "server")
	config.AddServer("10.1.0.172", "hera", "mediaExcel(0)", []string{}, "client")
	config.AddServer("10.1.0.174", "hera", "mediaExcel(0)", []string{}, "client")

	// 헬스 체크 실행
	report := config.RunCheck()
	fmt.Println("Health check completed!")
	fmt.Println(report)
}
