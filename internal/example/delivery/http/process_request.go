package http

import (
	"github.com/gin-gonic/gin"
)

// processCreateReq binds and validates the create item request body.
func (h *handler) processCreateReq(c *gin.Context) (createReq, error) {
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, err
	}
	return req, req.validate()
}

// processListReq binds and validates the list items query parameters.
func (h *handler) processListReq(c *gin.Context) (listReq, error) {
	var req listReq
	if err := c.ShouldBindQuery(&req); err != nil {
		return req, err
	}
	return req, req.validate()
}

// processUpdateReq binds and validates the update item request body + URI param.
func (h *handler) processUpdateReq(c *gin.Context) (updateReq, error) {
	var req updateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, err
	}
	req.ID = c.Param("id")
	if req.ID == "" {
		return req, gin.Error{}
	}
	return req, req.validate()
}
