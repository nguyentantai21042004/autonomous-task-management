package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NewOKResp returns a new OK response with the given data.
func NewOKResp(data any) Resp {
	return Resp{
		ErrorCode: 0,
		Message:   MessageSuccess,
		Data:      data,
	}
}

// OK sends 200 JSON with data.
func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, NewOKResp(data))
}

// Error sends error response with status code and message.
func Error(c *gin.Context, err error, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}

	c.JSON(http.StatusBadRequest, Resp{
		ErrorCode: 1,
		Message:   err.Error(),
		Data:      data,
	})
}

// InternalError sends 500 internal server error.
func InternalError(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, Resp{
		ErrorCode: InternalServerErrorCode,
		Message:   DefaultErrorMessage,
	})
}

// Unauthorized sends 401 response.
func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, Resp{
		ErrorCode: 401,
		Message:   "Unauthorized",
	})
}

// Forbidden sends 403 response.
func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, Resp{
		ErrorCode: 403,
		Message:   "Forbidden",
	})
}
