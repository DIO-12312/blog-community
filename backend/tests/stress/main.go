package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// StressTestConfig 压力测试配置
type StressTestConfig struct {
	BaseURL     string
	Concurrency int
	Duration    time.Duration
	Timeout     time.Duration
}

// RequestResult 单次请求结果
type RequestResult struct {
	StatusCode int
	Duration   time.Duration
	Error      error
}

// BenchmarkResult 基准测试汇总结果
type BenchmarkResult struct {
	Name        string
	TotalReqs   int64
	SuccessReqs int64
	FailReqs    int64
	MinLatency  time.Duration
	MaxLatency  time.Duration
	AvgLatency  time.Duration
	P50Latency  time.Duration
	P95Latency  time.Duration
	P99Latency  time.Duration
	RPS         float64
	Duration    time.Duration
	Latencies   []time.Duration
	Errors      map[string]int64
}

// =============================================================================
// HTTP 请求工具
// =============================================================================

func doRequest(method, url string, body []byte, headers map[string]string, timeout time.Duration) *RequestResult {
	client := &http.Client{Timeout: timeout}
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return &RequestResult{Error: err}
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &RequestResult{Duration: duration, Error: err}
	}
	defer resp.Body.Close()

	// 读取响应体
	io.ReadAll(resp.Body)

	return &RequestResult{
		StatusCode: resp.StatusCode,
		Duration:   duration,
	}
}

// =============================================================================
// 压力测试引擎
// =============================================================================

func runStressTest(name string, fn func() *RequestResult, config StressTestConfig) *BenchmarkResult {
	var (
		totalReqs   atomic.Int64
		successReqs atomic.Int64
		failReqs    atomic.Int64
		latencies   []time.Duration
		mu          sync.Mutex
		errorsMap   = make(map[string]int64)
	)

	// 工作协程
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	startTime := time.Now()

	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					result := fn()
					totalReqs.Add(1)

					mu.Lock()
					latencies = append(latencies, result.Duration)
					mu.Unlock()

					if result.Error != nil {
						failReqs.Add(1)
						errKey := result.Error.Error()
						if len(errKey) > 50 {
							errKey = errKey[:50]
						}
						mu.Lock()
						errorsMap[errKey]++
						mu.Unlock()
					} else if result.StatusCode >= 200 && result.StatusCode < 300 {
						successReqs.Add(1)
					} else {
						failReqs.Add(1)
						errKey := fmt.Sprintf("HTTP %d", result.StatusCode)
						mu.Lock()
						errorsMap[errKey]++
						mu.Unlock()
					}
				}
			}
		}()
	}

	// 运行指定时长
	time.Sleep(config.Duration)
	close(stopCh)
	wg.Wait()

	elapsed := time.Since(startTime)

	// 计算统计值
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	n := len(latencies)
	if n == 0 {
		return &BenchmarkResult{
			Name:     name,
			Duration: elapsed,
			Errors:   errorsMap,
		}
	}

	var sum int64
	for _, l := range latencies {
		sum += int64(l)
	}

	avg := time.Duration(sum / int64(n))
	min := latencies[0]
	max := latencies[n-1]
	p50 := latencies[n*50/100]
	p95 := latencies[n*95/100]
	p99 := latencies[n*99/100]

	return &BenchmarkResult{
		Name:        name,
		TotalReqs:   totalReqs.Load(),
		SuccessReqs: successReqs.Load(),
		FailReqs:    failReqs.Load(),
		MinLatency:  min,
		MaxLatency:  max,
		AvgLatency:  avg,
		P50Latency:  p50,
		P95Latency:  p95,
		P99Latency:  p99,
		RPS:         float64(totalReqs.Load()) / elapsed.Seconds(),
		Duration:    elapsed,
		Latencies:   latencies,
		Errors:      errorsMap,
	}
}

// =============================================================================
// 测试脚本定义
// =============================================================================

// 获取 JWT Token
func login(baseURL string) string {
	body := map[string]string{
		"username": "benchmark_user",
		"password": "benchmark123",
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(baseURL+"/api/users/login", "application/json", bytes.NewReader(data))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	var result struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	respBody, _ := io.ReadAll(resp.Body)
	json.Unmarshal(respBody, &result)
	return result.Data.Token
}

func main() {
	baseURL := flag.String("url", "http://localhost:8000", "API Gateway base URL")
	concurrency := flag.Int("c", 50, "Concurrency level")
	duration := flag.Int("d", 10, "Test duration in seconds")
	output := flag.String("o", "", "Output file (JSON format)")
	timeout := flag.Int("t", 30, "Request timeout in seconds")
	flag.Parse()

	config := StressTestConfig{
		BaseURL:     *baseURL,
		Concurrency: *concurrency,
		Duration:    time.Duration(*duration) * time.Second,
		Timeout:     time.Duration(*timeout) * time.Second,
	}

	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║         Blog Community — HTTP 压力测试工具               ║")
	fmt.Println("╠══════════════════════════════════════════════════════════╣")
	fmt.Printf("║  目标地址: %-46s ║\n", config.BaseURL)
	fmt.Printf("║  并发数:   %-46d ║\n", config.Concurrency)
	fmt.Printf("║  持续时间: %-46s ║\n", config.Duration)
	fmt.Printf("║  超时时间: %-46s ║\n", time.Duration(*timeout)*time.Second)
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// 先获取 token
	fmt.Print("[*] 正在获取认证 Token... ")
	token := login(config.BaseURL)
	if token != "" {
		fmt.Println("成功")
	} else {
		fmt.Println("失败（将在无认证状态下测试）")
	}
	fmt.Println()

	authHeaders := map[string]string{
		"Authorization": "Bearer " + token,
	}

	// 定义测试场景
	tests := []struct {
		Name   string
		Fn     func() *RequestResult
		Weight float64
	}{
		{
			Name: "GET /api/articles (文章列表)",
			Fn: func() *RequestResult {
				return doRequest("GET", config.BaseURL+"/api/articles?page=1&size=20", nil, nil, config.Timeout)
			},
		},
		{
			Name: "GET /api/articles/category/tech (分类文章)",
			Fn: func() *RequestResult {
				return doRequest("GET", config.BaseURL+"/api/articles/category/tech?page=1&size=20", nil, nil, config.Timeout)
			},
		},
		{
			Name: "GET /api/search?q=test (全文搜索)",
			Fn: func() *RequestResult {
				return doRequest("GET", config.BaseURL+"/api/search?q=test&page=1&size=20", nil, nil, config.Timeout)
			},
		},
		{
			Name: "POST /api/users/login (用户登录)",
			Fn: func() *RequestResult {
				body := map[string]string{
					"username": "benchmark_user",
					"password": "benchmark123",
				}
				data, _ := json.Marshal(body)
				return doRequest("POST", config.BaseURL+"/api/users/login", data, nil, config.Timeout)
			},
		},
		{
			Name: "GET /api/notifications/unread-count (未读通知数)",
			Fn: func() *RequestResult {
				return doRequest("GET", config.BaseURL+"/api/notifications/unread-count", nil, authHeaders, config.Timeout)
			},
		},
	}

	results := make([]*BenchmarkResult, 0, len(tests))

	// 运行单项测试
	fmt.Println("═══ 单项压力测试 ═══")
	fmt.Println()
	for _, test := range tests {
		fmt.Printf("[%s] 开始测试 (并发=%d, 时间=%s)...\n", test.Name, config.Concurrency, config.Duration)
		result := runStressTest(test.Name, test.Fn, config)
		results = append(results, result)
		printResult(result)
	}

	// 运行混合负载测试
	fmt.Println()
	fmt.Println("═══ 混合负载压力测试 (模拟真实流量) ═══")
	fmt.Println()

	mixedFns := []func() *RequestResult{
		// 40% 文章列表
		func() *RequestResult {
			return doRequest("GET", config.BaseURL+"/api/articles?page=1&size=20", nil, nil, config.Timeout)
		},
		func() *RequestResult {
			return doRequest("GET", config.BaseURL+"/api/articles?page=1&size=20", nil, nil, config.Timeout)
		},
		// 20% 搜索
		func() *RequestResult {
			return doRequest("GET", config.BaseURL+"/api/search?q=golang&page=1&size=20", nil, nil, config.Timeout)
		},
		// 15% 文章详情
		func() *RequestResult {
			return doRequest("GET", config.BaseURL+"/api/articles/sample-id", nil, nil, config.Timeout)
		},
		// 15% 分类浏览
		func() *RequestResult {
			return doRequest("GET", config.BaseURL+"/api/articles/category/tech?page=1&size=20", nil, nil, config.Timeout)
		},
		// 10% 通知查询
		func() *RequestResult {
			return doRequest("GET", config.BaseURL+"/api/notifications/unread-count", nil, authHeaders, config.Timeout)
		},
	}

	fmt.Printf("[混合负载] 开始测试 (并发=%d, 时间=%s)...\n", config.Concurrency, config.Duration)
	mixedResult := runStressTest("Mixed Workload (40%列表/20%搜索/15%详情/15%分类/10%通知)", func() *RequestResult {
		idx := time.Now().UnixNano() % int64(len(mixedFns))
		return mixedFns[idx]()
	}, config)
	results = append(results, mixedResult)
	printResult(mixedResult)

	// 输出汇总
	fmt.Println()
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println("                        汇总报告")
	fmt.Println("══════════════════════════════════════════════════════════════")
	fmt.Println()
	fmt.Printf("%-50s %8s %8s %8s %8s %8s %10s\n", "测试名称", "总请求", "成功", "失败", "P50", "P95", "RPS")
	fmt.Println(strings.Repeat("-", 105))
	for _, r := range results {
		fmt.Printf("%-50s %8d %8d %8d %8s %8s %9.1f\n",
			truncate(r.Name, 50),
			r.TotalReqs,
			r.SuccessReqs,
			r.FailReqs,
			formatDuration(r.P50Latency),
			formatDuration(r.P95Latency),
			r.RPS,
		)
	}
	fmt.Println()

	// 详细延迟分布
	fmt.Println()
	fmt.Println("═══ P50/P95/P99 延迟分布对比 ═══")
	fmt.Println()
	fmt.Printf("%-50s %10s %10s %10s %10s\n", "测试名称", "Min", "P50", "P95", "P99")
	fmt.Println(strings.Repeat("-", 95))
	for _, r := range results {
		if r.TotalReqs > 0 {
			fmt.Printf("%-50s %10s %10s %10s %10s\n",
				truncate(r.Name, 50),
				formatDuration(r.MinLatency),
				formatDuration(r.P50Latency),
				formatDuration(r.P95Latency),
				formatDuration(r.P99Latency),
			)
		}
	}

	// 输出 JSON
	if *output != "" {
		saveJSON(*output, results)
		fmt.Printf("\n[*] 详细结果已保存到: %s\n", *output)
	}
}

// =============================================================================
// 输出工具函数
// =============================================================================

func printResult(r *BenchmarkResult) {
	fmt.Printf("  ├─ 总请求数: %d\n", r.TotalReqs)
	fmt.Printf("  ├─ 成功/失败: %d / %d (成功率: %.2f%%)\n",
		r.SuccessReqs, r.FailReqs,
		float64(r.SuccessReqs)/float64(r.TotalReqs)*100)
	fmt.Printf("  ├─ RPS: %.1f req/s\n", r.RPS)
	fmt.Printf("  ├─ 平均延迟: %s\n", formatDuration(r.AvgLatency))
	fmt.Printf("  ├─ Min/Max: %s / %s\n", formatDuration(r.MinLatency), formatDuration(r.MaxLatency))
	fmt.Printf("  ├─ P50/P95/P99: %s / %s / %s\n",
		formatDuration(r.P50Latency), formatDuration(r.P95Latency), formatDuration(r.P99Latency))

	// 错误详情
	if len(r.Errors) > 0 {
		fmt.Println("  └─ 错误分布:")
		for err, count := range r.Errors {
			fmt.Printf("     • %s: %d (%.1f%%)\n",
				err, count, float64(count)/float64(r.TotalReqs)*100)
		}
	} else {
		fmt.Println("  └─ 无错误")
	}
	fmt.Println()
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "-"
	}
	if d < time.Microsecond {
		return fmt.Sprintf("%.2fns", float64(d.Nanoseconds()))
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fus", float64(d.Microseconds()))
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func saveJSON(filename string, results []*BenchmarkResult) {
	type jsonResult struct {
		Name        string            `json:"name"`
		TotalReqs   int64             `json:"total_reqs"`
		SuccessReqs int64             `json:"success_reqs"`
		FailReqs    int64             `json:"fail_reqs"`
		SuccessRate float64           `json:"success_rate"`
		MinLatency  string            `json:"min_latency"`
		MaxLatency  string            `json:"max_latency"`
		AvgLatency  string            `json:"avg_latency"`
		P50Latency  string            `json:"p50_latency"`
		P95Latency  string            `json:"p95_latency"`
		P99Latency  string            `json:"p99_latency"`
		RPS         float64           `json:"rps"`
		Duration    string            `json:"duration"`
		Errors      map[string]int64  `json:"errors,omitempty"`
		StdDev      string            `json:"std_dev"`
	}

	var out []jsonResult
	for _, r := range results {
		stdDev := calcStdDev(r.Latencies, r.AvgLatency)
		out = append(out, jsonResult{
			Name:        r.Name,
			TotalReqs:   r.TotalReqs,
			SuccessReqs: r.SuccessReqs,
			FailReqs:    r.FailReqs,
			SuccessRate: float64(r.SuccessReqs) / float64(r.TotalReqs) * 100,
			MinLatency:  formatDuration(r.MinLatency),
			MaxLatency:  formatDuration(r.MaxLatency),
			AvgLatency:  formatDuration(r.AvgLatency),
			P50Latency:  formatDuration(r.P50Latency),
			P95Latency:  formatDuration(r.P95Latency),
			P99Latency:  formatDuration(r.P99Latency),
			RPS:         r.RPS,
			Duration:    r.Duration.String(),
			Errors:      r.Errors,
			StdDev:      formatDuration(stdDev),
		})
	}

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("[!] 无法创建输出文件: %v\n", err)
		return
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		fmt.Printf("[!] JSON 编码失败: %v\n", err)
	}
}

func calcStdDev(latencies []time.Duration, mean time.Duration) time.Duration {
	if len(latencies) < 2 {
		return 0
	}
	var sumSquares float64
	meanFloat := float64(mean)
	for _, l := range latencies {
		diff := float64(l) - meanFloat
		sumSquares += diff * diff
	}
	variance := sumSquares / float64(len(latencies))
	return time.Duration(math.Sqrt(variance))
}
