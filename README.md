# 一、说明

[![codecov](https://codecov.io/gh/ivfzhou/goroutine-pool/graph/badge.svg?token=N949TSNU2T)](https://codecov.io/gh/ivfzhou/goroutine-pool)
[![Go Reference](https://pkg.go.dev/badge/github.com/ivfzhou/goroutine-pool.svg)](https://pkg.go.dev/github.com/ivfzhou/goroutine-pool)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivfzhou/goroutine-pool)](https://goreportcard.com/report/github.com/ivfzhou/goroutine-pool)

高性能的 Go 协程池，支持自动扩缩容、任务队列缓冲和优雅关闭。

# 二、特性

- **自动扩缩容**：根据负载自动调整协程数量，支持配置最小/最大协程数
- **任务队列缓冲**：可配置等待队列，避免任务丢失
- **空闲回收**：自动回收空闲协程，节省系统资源
- **优雅关闭**：等待所有任务完成后安全关闭
- **Panic 恢复**：自动捕获并记录任务中的 Panic，防止协程崩溃
- **线程安全**：所有操作都是并发安全的
- **高性能**：基于 Lock-Free 队列实现，减少锁竞争

# 三、可应用场景

- **批量任务处理**：需要并发执行大量耗时相近的短任务（如批量导出、批量计算、图片/视频转码、文件处理），通过协程池统一调度，避免为每个任务都创建协程带来的开销。
- **并发请求聚合**：同时调用多个下游接口、微服务或数据库（如批量 RPC、HTTP 请求、批量 SQL 查询），控制并发度，防止瞬间打爆后端资源。
- **消息/队列消费**：作为消息队列或任务队列的消费者，持续拉取消息并交给协程池处理，结合 `WithMaxWaitingSize` 配置缓冲队列，实现削峰填谷。
- **流量控制与限流**：通过 `WithMaxSize` 限制最大并发协程数，为爬虫、外部 API 调用等场景提供天然的并发上限保护。
- **需要优雅关闭的服务**：在 HTTP 服务、定时任务或常驻进程中，用 `Close()` 等待所有在途任务完成后安全退出，避免任务被中断或丢失。
- **需要动态扩缩容的负载**：流量波动明显的场景（如秒杀、活动高峰），协程池可根据负载自动增减协程，并在空闲时回收，节省系统资源。

# 四、安装

```bash
go get github.com/ivfzhou/goroutine-pool@latest
```

# 五、快速开始

```go
package main

import (
    "fmt"
    "time"
    pool "github.com/ivfzhou/goroutine-pool"
)

func main() {
    // 创建协程池
    p := pool.New(
        pool.WithInitSize(10),      // 初始协程数：10
        pool.WithMaxSize(100),      // 最大协程数：100
        pool.WithMinIdleSize(5),    // 最小空闲协程数：5
        pool.WithMaxIdleTimeout(30 * time.Second), // 空闲超时：30秒
    )
    defer p.Close()

    // 提交任务
    for i := 0; i < 100; i++ {
        taskID := i
        err := p.Submit(func() {
            fmt.Printf("Task %d is running\n", taskID)
            time.Sleep(100 * time.Millisecond)
        })
        
        if err != nil {
            fmt.Printf("Task %d failed: %v\n", taskID, err)
        }
    }

    // 查看协程池状态
    fmt.Printf("Worker size: %d\n", p.WorkerSize())
    fmt.Printf("Waiting tasks: %d\n", p.WaitingTaskSize())
}
```
