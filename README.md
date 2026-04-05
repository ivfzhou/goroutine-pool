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

# 三、安装

```bash
go get github.com/ivfzhou/goroutine-pool@latest
```

# 四、快速开始

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

# 五、配置选项

```go
// 设置协程池最大容量（默认：math.MaxUint32）
pool.WithMaxSize(1000)

// 设置最小空闲协程数（默认：等于初始协程数）
pool.WithMinIdleSize(50)

// 设置初始协程数（默认：0）
pool.WithInitSize(10)

// 设置协程最大空闲时间（默认：5秒）
pool.WithMaxIdleTimeout(30 * time.Second)

// 设置任务等待队列最大长度（默认：0，不启用队列）
pool.WithMaxWaitingSize(10000)
```

# 六、完整示例

## 6.1 示例 1：使用任务队列缓冲

```go
p := pool.New(
    pool.WithMaxSize(10),
    pool.WithMaxWaitingSize(1000), // 启用任务队列，最多缓存1000个任务
)
defer p.Close()

// 即使瞬间提交大量任务，也不会丢失
for i := 0; i < 10000; i++ {
    err := p.Submit(func() {
        // 任务逻辑
    })
    
    if err != nil {
        // 队列满了或池已关闭
        log.Printf("Submit failed: %v", err)
    }
}
```

## 6.2 示例 2：优雅关闭

```go
p := pool.New(pool.WithMaxSize(100))

// 提交一些长时间运行的任务
for i := 0; i < 50; i++ {
    p.Submit(func() {
        time.Sleep(5 * time.Second)
        fmt.Println("Task completed")
    })
}

// 等待所有任务完成后关闭
p.Close()
fmt.Println("Pool closed gracefully")
```

## 6.3 示例 3：监控协程池状态

```go
p := pool.New(
    pool.WithInitSize(10),
    pool.WithMaxSize(100),
    pool.WithMaxIdleTimeout(10 * time.Second),
)

go func() {
    ticker := time.NewTicker(1 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        fmt.Printf("Workers: %d, Waiting tasks: %d\n", 
            p.WorkerSize(), 
            p.WaitingTaskSize())
    }
}()

// 使用协程池...
```

# 七、API 文档

## 7.1 创建协程池

```go
func New(options ...OptionFunc) *Pool
```

创建一个新的协程池，支持通过选项进行配置。

## 7.2 提交任务

```go
func (p *Pool) Submit(fn func()) error
```

提交一个任务到协程池，返回可能的错误：
- `ErrPoolIsClosed`：协程池已关闭
- `ErrPoolIsOverload`：协程池过载（无空闲协程且达到最大协程数，且队列已满）

## 7.3 关闭协程池

```go
func (p *Pool) Close()
```

优雅地关闭协程池，等待所有任务完成后退出。

## 7.4 查询状态

```go
// 获取当前运行的协程数量
func (p *Pool) WorkerSize() int

// 获取等待队列中的任务数量
func (p *Pool) WaitingTaskSize() int

// 获取协程池最大容量
func (p *Pool) Cap() int

// 判断协程池是否已关闭
func (p *Pool) IsClosed() bool
```

# 八、错误处理

```go
var (
    // 协程池已满，无法提交新任务
    ErrPoolIsOverload = errors.New("pool is overload")
    
    // 协程池已关闭
    ErrPoolIsClosed = errors.New("pool is closed")
)
```
