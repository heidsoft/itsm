package controller

import (
	"github.com/gin-gonic/gin"
	"itsm-backend/common"
	"itsm-backend/dto"
	"itsm-backend/middleware"
	"time"
)

type AuthController struct {
	jwtSecret string
}

func NewAuthController(jwtSecret string) *AuthController {
	return &AuthController{
		jwtSecret: jwtSecret,
	}
}

// Login 用户登录
func (ac *AuthController) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, 1001, "参数错误: "+err.Error())
		return
	}

	// 简单的用户验证（实际项目中应该查询数据库）
	if req.Username == "admin" && req.Password == "admin123" {
		// 生成JWT token
		token, err := middleware.GenerateToken(
			1,       // userID
			"admin", // username
			"admin", // role
			ac.jwtSecret,
			time.Hour*24*7, // 7天过期
		)
		if err != nil {
			common.Fail(c, 5001, "生成token失败")
			return
		}

		response := dto.LoginResponse{
			Token: token,
			User: dto.UserInfo{
				ID:       1,
				Username: "admin",
				Role:     "admin",
			},
		}
		common.Success(c, response)
	} else {
		common.Fail(c, 2001, "用户名或密码错误")
	}
}
