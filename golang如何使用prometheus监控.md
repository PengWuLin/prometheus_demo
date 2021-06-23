# 使用 Prometheus 对 Go 应用程序进行监测

Prometheus提供了 Golang 的插桩库，可以用来注册，收集和暴露服务的指标。

## 指标类型

一共四种指标类型。

### Counter（计数器）

counter是一个累计的指标，代表一个单调递增的计数器，它的值只会增加或在重启时重置为零。例如，可以使用 counter 来监控服务器登录次数，消息队列的处理数量。

### Gauge（计量器）

gauge是代表一个数值类型的指标，它的值可以增或减。gauge 通常用于一些度量的值，例如当前CPU温度，CPU使用率，内存使用率；也可以用于一些可以增减的“计数”，如当前服务 goroutine 个数。

### Histogram（分布图）

histogram 对观测值（类似请求延迟或回复包大小）进行采样，并用一些可配置的桶来计数。它也会给出一个所有观测值的总和。

### Summary（摘要）

跟histogram 类似，summary也对观测值（类似请求延迟或回复包大小）进行采样。同时它会给出一个总数以及所有观测值的总和，它在一个滑动的时间窗口上计算可配置的分位数。

## golang服务器加入prometheus采集指标步骤说明

### 定义指标

对需要进行采集的数据，选择符合的指标类型，并定义指标。如登录次数使用counter，cpu与mem使用率用gauge。

同时可以对一些指标加入label，如下面的login_count，可以加入label参数，来统计不同ip登录次数。

```go
var (
	loginCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "login_count",
			Help: "cpu使用率",
		},
		[]string{"client_ip"},
	)

	cpuPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_percent",
			Help: "cpu使用率",
		},
	)

	memUsedPercent = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mem_used_percent",
			Help: "内存使用率",
		},
	)
)
```

### 注册指标

一般指标注册写在init函数中

```go
func init() {
   prometheus.MustRegister(loginCount, cpuPercent, memUsedPercent)
}
```

### 打桩

登录次数这种一般嵌在http的handler中，cpu和mem的采集，则可以单独开goroutine进行采集。

这里http服务使用gin框架。

监控登录函数次数

```
// 匹配路由 login?firstname=wulin&lastname=peng
engine.GET("/login", func(c *gin.Context) {
   ip := c.ClientIP()
   loginCount.WithLabelValues(ip).Inc()

   firstname := c.DefaultQuery("firstname", "Guest")
   lastname := c.Query("lastname")
   c.String(http.StatusOK, "Hello %s %s", firstname, lastname)
})
```

 监控cpu与mem状态，这里使用随机数模拟数据

```
func collect() {
   tk := time.NewTicker(time.Second * 5)
   defer tk.Stop()
   for {
      select {
      case <-tk.C:
         cpuPercent.Set(GetCpuPercent())
         memUsedPercent.Set(GetMemPercent())
      }
   }
}

func GetCpuPercent() float64 {
   return float64(randInt(0, 100))
}

func GetMemPercent() float64 {
   return float64(randInt(0, 100))
}

func randInt(min, max int) int {
   return min + rand.Intn(max-min)
}
```

### http服务暴露采集路由

prometheus采集数据是使用pull方式，在prometheus中配置待采集服务器的地址，采集周期，采集路由等，同时http服务需要暴露一个采集接口。

这里使用默认的路由。

```
engine.GET("/metrics", func(c *gin.Context) {
   promhttp.Handler().ServeHTTP(c.Writer, c.Request)
   return
})
```

我们的demo监听10000端口，和prometheus跑在同一台机器上，prometheus配置如下

```
- job_name: 'collect'
  # metrics_path defaults to '/metrics'
  # scheme defaults to 'http'.
  static_configs:
  - targets: ['127.0.0.1:10000']
```

这时候重启下prometheus，浏览器打开 http://192.168.15.129:9090/targets 可以看到prometheus成功监控到我们的服务。

![1624466183933](picture\targets.png)





还可以使用prometheus自带的graph，进行绘图。

![1624466183933](picture\graph.png)



## demo源码

```
package main

import (
   "flag"
   "math/rand"
   "net/http"
   "time"

   "github.com/gin-gonic/gin"
   "github.com/prometheus/client_golang/prometheus"
   "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
   loginCount = prometheus.NewCounterVec(
      prometheus.CounterOpts{
         Name: "login_count",
         Help: "cpu使用率",
      },
      []string{"client_ip"},
   )

   cpuPercent = prometheus.NewGauge(
      prometheus.GaugeOpts{
         Name: "cpu_percent",
         Help: "cpu使用率",
      },
   )

   memUsedPercent = prometheus.NewGauge(
      prometheus.GaugeOpts{
         Name: "mem_used_percent",
         Help: "内存使用率",
      },
   )
)

func init() {
   prometheus.MustRegister(loginCount, cpuPercent, memUsedPercent)
}

var address = flag.String("address", "0.0.0.0:10000", "服务器监听地址")

func main() {

   flag.Parse()

   go collect()

   engine := gin.New()
   engine.GET("/metrics", func(c *gin.Context) {
      promhttp.Handler().ServeHTTP(c.Writer, c.Request)
      return
   })

   // 匹配路由 login?firstname=wulin&lastname=peng
   engine.GET("/login", func(c *gin.Context) {
      ip := c.ClientIP()
      loginCount.WithLabelValues(ip).Inc()

      firstname := c.DefaultQuery("firstname", "Guest")
      lastname := c.Query("lastname")
      c.String(http.StatusOK, "Hello %s %s", firstname, lastname)
   })

   err := engine.Run(*address)
   if err != nil {
      panic(err)
   }
}

func collect() {
   tk := time.NewTicker(time.Second * 5)
   defer tk.Stop()
   for {
      select {
      case <-tk.C:
         cpuPercent.Set(GetCpuPercent())
         memUsedPercent.Set(GetMemPercent())
      }
   }
}

func GetCpuPercent() float64 {
   return float64(randInt(0, 100))
}

func GetMemPercent() float64 {
   return float64(randInt(0, 100))
}

func randInt(min, max int) int {
   return min + rand.Intn(max-min)
}
```

