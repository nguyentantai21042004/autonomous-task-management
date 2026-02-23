package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/pkg/response"
)

// Create godoc
// @Summary     Create a new item
// @Description Creates a new item with the provided name and description.
// @Tags        Example
// @Accept      json
// @Produce     json
// @Param       body body createReq true "Item data"
// @Success     200  {object} createResp
// @Failure     400  {object} response.Resp "Bad Request"
// @Failure     409  {object} response.Resp "Conflict - name already exists"
// @Failure     500  {object} response.Resp "Internal Server Error"
// @Router      /api/v1/example/items [POST]
func (h *handler) Create(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processCreateReq(c)
	if err != nil {
		response.Error(c, err, h.discord)
		return
	}

	output, err := h.uc.Create(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "uc.Create: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newCreateResp(output))
}

// List godoc
// @Summary     List items
// @Description Returns a paginated list of items with optional status filter.
// @Tags        Example
// @Accept      json
// @Produce     json
// @Param       status query string false "Filter by status (active/inactive)"
// @Param       limit  query int    false "Page size (default: 20)"
// @Param       offset query int    false "Page offset (default: 0)"
// @Success     200 {object} listResp
// @Failure     400 {object} response.Resp "Bad Request"
// @Failure     500 {object} response.Resp "Internal Server Error"
// @Router      /api/v1/example/items [GET]
func (h *handler) List(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processListReq(c)
	if err != nil {
		response.Error(c, err, h.discord)
		return
	}

	output, err := h.uc.List(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "uc.List: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newListResp(output))
}

// Detail godoc
// @Summary     Get item detail
// @Description Returns a single item by its ID.
// @Tags        Example
// @Accept      json
// @Produce     json
// @Param       id path string true "Item ID"
// @Success     200 {object} detailResp
// @Failure     404 {object} response.Resp "Not Found"
// @Failure     500 {object} response.Resp "Internal Server Error"
// @Router      /api/v1/example/items/{id} [GET]
func (h *handler) Detail(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	output, err := h.uc.Detail(ctx, id)
	if err != nil {
		h.l.Errorf(ctx, "uc.Detail: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newDetailResp(output))
}

// Update godoc
// @Summary     Update an item
// @Description Updates an existing item. All fields are optional (partial update).
// @Tags        Example
// @Accept      json
// @Produce     json
// @Param       id   path string    true "Item ID"
// @Param       body body updateReq true "Fields to update"
// @Success     200 {object} updateResp
// @Failure     400 {object} response.Resp "Bad Request"
// @Failure     404 {object} response.Resp "Not Found"
// @Failure     500 {object} response.Resp "Internal Server Error"
// @Router      /api/v1/example/items/{id} [PUT]
func (h *handler) Update(c *gin.Context) {
	ctx := c.Request.Context()

	req, err := h.processUpdateReq(c)
	if err != nil {
		response.Error(c, err, h.discord)
		return
	}

	output, err := h.uc.Update(ctx, req.toInput())
	if err != nil {
		h.l.Errorf(ctx, "uc.Update: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, h.newUpdateResp(output))
}

// Delete godoc
// @Summary     Delete an item
// @Description Permanently removes an item by ID.
// @Tags        Example
// @Accept      json
// @Produce     json
// @Param       id path string true "Item ID"
// @Success     200 {object} response.Resp "OK"
// @Failure     404 {object} response.Resp "Not Found"
// @Failure     500 {object} response.Resp "Internal Server Error"
// @Router      /api/v1/example/items/{id} [DELETE]
func (h *handler) Delete(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	if err := h.uc.Delete(ctx, id); err != nil {
		h.l.Errorf(ctx, "uc.Delete: %v", err)
		response.Error(c, h.mapError(err), h.discord)
		return
	}

	response.OK(c, nil)
}
