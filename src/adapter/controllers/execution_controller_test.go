package controllers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"mechmanager-execution/adapter/controllers"
	"mechmanager-execution/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- mock use case ---

type mockUseCase struct {
	executions []*domain.Execution
	err        error
}

func (m *mockUseCase) CreateFromOrder(orderID, mechanicID string) (*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &domain.Execution{ID: uuid.New(), OrderID: orderID}, nil
}

func (m *mockUseCase) FindByID(id uuid.UUID) (*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.executions {
		if e.ID == id {
			return e, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUseCase) FindByOrderID(orderID string) (*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.executions {
		if e.OrderID == orderID {
			return e, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUseCase) ListAll() ([]*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.executions, nil
}

func (m *mockUseCase) UpdateStatus(id uuid.UUID, status domain.ExecutionStatus, notes string) (*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.executions {
		if e.ID == id {
			e.Status = status
			return e, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUseCase) Complete(id uuid.UUID, repairNotes string) (*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.executions {
		if e.ID == id {
			e.Status = domain.ExecutionStatusCompleted
			e.RepairNotes = repairNotes
			return e, nil
		}
	}
	return nil, errors.New("not found")
}

func (m *mockUseCase) Fail(id uuid.UUID, reason string) (*domain.Execution, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, e := range m.executions {
		if e.ID == id {
			e.Status = domain.ExecutionStatusFailed
			return e, nil
		}
	}
	return nil, errors.New("not found")
}

// --- helpers ---

func setupRouter(uc *mockUseCase) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ctrl := controllers.NewExecutionController(uc)
	r.GET("/executions", ctrl.ListAll)
	r.GET("/executions/:id", ctrl.FindByID)
	r.GET("/executions/order/:id", ctrl.FindByOrderID)
	r.PATCH("/executions/:id/status", ctrl.UpdateStatus)
	r.PATCH("/executions/:id/complete", ctrl.Complete)
	r.PATCH("/executions/:id/fail", ctrl.Fail)
	return r
}

// --- ListAll ---

func TestListAll_Success(t *testing.T) {
	id := uuid.New()
	uc := &mockUseCase{executions: []*domain.Execution{{ID: id, OrderID: "order-1", Status: domain.ExecutionStatusQueued}}}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions", nil))

	if w.Code != http.StatusOK {
		t.Errorf("esperado 200, got %d", w.Code)
	}
}

func TestListAll_Error(t *testing.T) {
	uc := &mockUseCase{err: errors.New("db error")}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions", nil))

	if w.Code != http.StatusInternalServerError {
		t.Errorf("esperado 500, got %d", w.Code)
	}
}

// --- FindByID ---

func TestFindByID_Success(t *testing.T) {
	id := uuid.New()
	uc := &mockUseCase{executions: []*domain.Execution{{ID: id, OrderID: "order-1", Status: domain.ExecutionStatusQueued}}}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions/"+id.String(), nil))

	if w.Code != http.StatusOK {
		t.Errorf("esperado 200, got %d", w.Code)
	}
}

func TestFindByID_InvalidUUID(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions/not-a-uuid", nil))

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	uc := &mockUseCase{executions: []*domain.Execution{}}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions/"+uuid.New().String(), nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("esperado 404, got %d", w.Code)
	}
}

// --- FindByOrderID ---

func TestFindByOrderID_Success(t *testing.T) {
	id := uuid.New()
	uc := &mockUseCase{executions: []*domain.Execution{{ID: id, OrderID: "order-abc", Status: domain.ExecutionStatusQueued}}}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions/order/order-abc", nil))

	if w.Code != http.StatusOK {
		t.Errorf("esperado 200, got %d", w.Code)
	}
}

func TestFindByOrderID_NotFound(t *testing.T) {
	uc := &mockUseCase{executions: []*domain.Execution{}}
	r := setupRouter(uc)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/executions/order/inexistente", nil))

	if w.Code != http.StatusNotFound {
		t.Errorf("esperado 404, got %d", w.Code)
	}
}

// --- UpdateStatus ---

func TestUpdateStatus_Success(t *testing.T) {
	id := uuid.New()
	uc := &mockUseCase{executions: []*domain.Execution{{ID: id, OrderID: "order-1", Status: domain.ExecutionStatusQueued}}}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"status": "IN_DIAGNOSIS", "diagnostic_notes": "barulho"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/"+id.String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperado 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestUpdateStatus_InvalidUUID(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"status": "IN_DIAGNOSIS"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/not-uuid/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestUpdateStatus_InvalidJSON(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	req := httptest.NewRequest(http.MethodPatch, "/executions/"+uuid.New().String()+"/status", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestUpdateStatus_UseCaseError(t *testing.T) {
	uc := &mockUseCase{err: errors.New("transição inválida")}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"status": "IN_DIAGNOSIS"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/"+uuid.New().String()+"/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

// --- Complete ---

func TestComplete_Success(t *testing.T) {
	id := uuid.New()
	uc := &mockUseCase{executions: []*domain.Execution{{ID: id, OrderID: "order-1", Status: domain.ExecutionStatusInRepair}}}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"repair_notes": "correia trocada"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/"+id.String()+"/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperado 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestComplete_InvalidUUID(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"repair_notes": "notas"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/not-uuid/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestComplete_InvalidJSON(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	req := httptest.NewRequest(http.MethodPatch, "/executions/"+uuid.New().String()+"/complete", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestComplete_UseCaseError(t *testing.T) {
	uc := &mockUseCase{err: errors.New("não está em IN_REPAIR")}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"repair_notes": "notas"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/"+uuid.New().String()+"/complete", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

// --- Fail ---

func TestFail_Success(t *testing.T) {
	id := uuid.New()
	uc := &mockUseCase{executions: []*domain.Execution{{ID: id, OrderID: "order-1", Status: domain.ExecutionStatusQueued}}}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"reason": "peça indisponível"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/"+id.String()+"/fail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("esperado 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestFail_InvalidUUID(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"reason": "motivo"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/not-uuid/fail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestFail_InvalidJSON(t *testing.T) {
	uc := &mockUseCase{}
	r := setupRouter(uc)

	req := httptest.NewRequest(http.MethodPatch, "/executions/"+uuid.New().String()+"/fail", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}

func TestFail_UseCaseError(t *testing.T) {
	uc := &mockUseCase{err: errors.New("falha interna")}
	r := setupRouter(uc)

	body, _ := json.Marshal(map[string]string{"reason": "motivo"})
	req := httptest.NewRequest(http.MethodPatch, "/executions/"+uuid.New().String()+"/fail", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("esperado 400, got %d", w.Code)
	}
}
