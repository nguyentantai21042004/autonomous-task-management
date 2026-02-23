package http

import (
	"time"

	"autonomous-task-management/internal/example"
)

// --- Request DTOs ---

type createReq struct {
	Name        string `json:"name"        binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
}

func (r createReq) validate() error { return nil }

func (r createReq) toInput() example.CreateItemInput {
	return example.CreateItemInput{
		Name:        r.Name,
		Description: r.Description,
	}
}

// ---

type listReq struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
}

func (r listReq) validate() error { return nil }

func (r listReq) toInput() example.ListItemsInput {
	limit := r.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if r.Offset < 0 {
		r.Offset = 0
	}
	return example.ListItemsInput{
		Status: r.Status,
		Limit:  limit,
		Offset: r.Offset,
	}
}

// ---

type updateReq struct {
	ID          string `json:"-"` // populated from URI param
	Name        string `json:"name"        binding:"omitempty,min=1,max=255"`
	Description string `json:"description" binding:"omitempty,max=1000"`
	Status      string `json:"status"      binding:"omitempty,oneof=active inactive"`
}

func (r updateReq) validate() error { return nil }

func (r updateReq) toInput() example.UpdateItemInput {
	return example.UpdateItemInput{
		ID:          r.ID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
	}
}

// --- Response DTOs ---

type itemResp struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func newItemResp(item example.Item) itemResp {
	return itemResp{
		ID:          item.ID,
		Name:        item.Name,
		Description: item.Description,
		Status:      item.Status,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

type createResp struct {
	Item itemResp `json:"item"`
}

func (h *handler) newCreateResp(out example.CreateItemOutput) createResp {
	return createResp{Item: newItemResp(out.Item)}
}

type listResp struct {
	Items  []itemResp `json:"items"`
	Total  int        `json:"total"`
	Limit  int        `json:"limit"`
	Offset int        `json:"offset"`
}

func (h *handler) newListResp(out example.ListItemsOutput) listResp {
	items := make([]itemResp, len(out.Items))
	for i, item := range out.Items {
		items[i] = newItemResp(item)
	}
	return listResp{
		Items:  items,
		Total:  out.Total,
		Limit:  out.Limit,
		Offset: out.Offset,
	}
}

type detailResp struct {
	Item itemResp `json:"item"`
}

func (h *handler) newDetailResp(out example.DetailItemOutput) detailResp {
	return detailResp{Item: newItemResp(out.Item)}
}

type updateResp struct {
	Item itemResp `json:"item"`
}

func (h *handler) newUpdateResp(out example.UpdateItemOutput) updateResp {
	return updateResp{Item: newItemResp(out.Item)}
}
