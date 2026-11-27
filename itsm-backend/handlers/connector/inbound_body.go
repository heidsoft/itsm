package connector

import (
	"errors"
	"io"
	"net/http"

	"itsm-backend/common"

	"github.com/gin-gonic/gin"
)

const maxInboundCallbackBytes = 1 << 20

// ReadInboundCallbackBody returns the complete raw body for signature verification.
func ReadInboundCallbackBody(c *gin.Context) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxInboundCallbackBytes))
	if err == nil {
		return body, true
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, common.Response{
			Code: common.ParamErrorCode, Message: "回调请求体超过 1 MiB",
		})
		return nil, false
	}
	common.Fail(c, common.ParamErrorCode, "无法读取回调请求体")
	return nil, false
}
