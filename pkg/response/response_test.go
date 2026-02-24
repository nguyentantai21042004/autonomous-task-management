package response_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/pkg/response"
)

func TestResponses(t *testing.T) {
	// Setup Gin test mode
	gin.SetMode(gin.TestMode)

	t.Run("OK", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		data := map[string]string{"foo": "bar"}
		response.OK(c, data)

		if w.Code != http.StatusOK {
			t.Errorf("expected %d but got %d", http.StatusOK, w.Code)
		}

		var resp response.Resp
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}

		if resp.ErrorCode != 0 {
			t.Errorf("expected ErrorCode 0, got %d", resp.ErrorCode)
		}
		// Data is mapped as float64/string/map depending on unmarshal
		dMap, ok := resp.Data.(map[string]interface{})
		if !ok || dMap["foo"] != "bar" {
			t.Errorf("unexpected data payload: %v", resp.Data)
		}
	})

	t.Run("Error", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		errData := map[string]interface{}{"field": "invalid"}
		response.Error(c, errors.New("test err"), errData)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected %d, got %d", http.StatusBadRequest, w.Code)
		}

		var resp response.Resp
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp.ErrorCode != 1 {
			t.Errorf("expected ErrorCode 1, got %d", resp.ErrorCode)
		}
		if resp.Message != "test err" {
			t.Errorf("expected message 'test err', got %s", resp.Message)
		}
	})

	t.Run("Error Nil Data", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		response.Error(c, errors.New("test err nil"), nil)

		var resp response.Resp
		json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data == nil {
			t.Errorf("expected empty map for nil data, got nil")
		}
	})

	t.Run("InternalError", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		response.InternalError(c, errors.New("db crash"))

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", w.Code)
		}
	})

	t.Run("Unauthorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		response.Unauthorized(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", w.Code)
		}
	})

	t.Run("Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		response.Forbidden(c)

		if w.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", w.Code)
		}
	})
}
