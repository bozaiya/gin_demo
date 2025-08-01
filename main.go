package main

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

type Option struct {
	ID    int    `json:"id"`
	Text  string `json:"text"`
	Votes int    `json:"votes"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("mysql", "root:123456@tcp(localhost:3306)/vote_system")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	router := gin.Default()
	router.Use(CORSMiddleware())

	router.GET("/api/poll", getPoll)
	router.POST("/api/poll/vote", postVote)
	router.GET("/ws/poll", handleWebSocket)

	router.Run(":8080")
}

// 中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置允许跨域访问的源
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		// 允许的HTTP方法
		c.Writer.Header().Set("Access-Control-Allow-Methods", "*")
		// 允许的请求头
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		// 处理预检请求（OPTIONS）
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204) // 预检请求直接返回 204
			return
		}
		c.Next() // 继续处理后续中间件或路由
	}
}

func getPoll(c *gin.Context) {
	rows, err := db.Query("SELECT id, text, votes FROM options")
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var options []Option
	for rows.Next() {
		var opt Option
		if err := rows.Scan(&opt.ID, &opt.Text, &opt.Votes); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		options = append(options, opt)
	}

	// 返回JSON数据
	c.JSON(http.StatusOK, gin.H{
		"question": "您最喜欢哪个选项？",
		"options":  options,
	})
}

func postVote(c *gin.Context) {
	// 解析请求体中的 optionId
	var req struct {
		OptionID int `json:"optionId"`
	}
	// 绑定 JSON 数据
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效请求"})
		return
	}

	// 更新数据库票数
	_, err := db.Exec("UPDATE options SET votes = votes + 1 WHERE id = ?", req.OptionID)
	if err != nil {
		c.JSON(500, gin.H{"error": "投票失败"})
		return
	}

	// 异步广播更新
	go broadcastUpdate()
	c.JSON(200, gin.H{"message": "投票成功"})
}

// 存储所有 WebSocket 客户端
var clients = make(map[*websocket.Conn]bool)

// 保护 clients 的互斥锁
var clientsMutex sync.Mutex

// WebSocket配置
var upgrader = websocket.Upgrader{
	// 允许跨域
	CheckOrigin: func(r *http.Request) bool { return true },
}

// 处理websocket
func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	for {
		if _, _, err := conn.NextReader(); err != nil {
			clientsMutex.Lock()
			delete(clients, conn)
			clientsMutex.Unlock()
			break
		}
	}
}

func broadcastUpdate() {
	// 查询最新数据
	rows, err := db.Query("SELECT id, text, votes FROM options")
	if err != nil {
		return
	}
	defer rows.Close()

	var options []Option
	for rows.Next() {
		var opt Option
		if err := rows.Scan(&opt.ID, &opt.Text, &opt.Votes); err != nil {
			return
		}
		options = append(options, opt)
	}

	// 构建推送数据
	data := gin.H{
		"question": "您最喜欢哪个选项？",
		"options":  options,
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	// 遍历所有客户端
	for client := range clients {
		if err := client.WriteJSON(data); err != nil { // 发送 JSON 数据
			client.Close()
			delete(clients, client)
		}
	}
}
