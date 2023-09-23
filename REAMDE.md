### 模块说明

go协程管理池

### 快速开始

```golang
import pool "github.com/ivfzhou/goroutine-pool"

// 创建协程池
p := pool.New(pool.WithInitSize(1024), pool.WithMinIdleSize(1024), pool.WithMaxWaitingSize(1024))

// 提交任务
p.Submit(func (){})

// 查看协程数量
p.WorkerSize()

// 查看阻塞的任务数
p.WaitingTaskSize()

// 关闭协程池
pool.Close()
```

联系电邮：ivfzhou@126.com
