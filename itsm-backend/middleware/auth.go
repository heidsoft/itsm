package middleware

import (
	"strings"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			common.Fail(c, common.AuthFailedCode, "缺少认证token")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		if !strings.HasPrefix(authHeader, "Bearer ") {
			common.Fail(c, common.AuthFailedCode, "token格式错误")
			c.Abort()
			return
		}

		// 提取token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			common.Fail(c, common.AuthFailedCode, "token不能为空")
			c.Abort()
			return
		}

		// TODO: 这里应该验证JWT token并解析用户信息
		// 现在为了演示，我们模拟一个用户ID
		// 在实际项目中，你需要:
		// 1. 验证JWT token的有效性
		// 2. 解析token中的用户信息
		// 3. 可选：从数据库验证用户状态
		
		// 模拟用户ID（实际应从JWT中解析）
		userID := 1 // 这里应该从JWT token中解析出真实的用户ID
		
		// 将用户ID存储到上下文中
		c.Set("user_id", userID)
		c.Set("token", token)

		c.Next()
	}
}