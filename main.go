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
