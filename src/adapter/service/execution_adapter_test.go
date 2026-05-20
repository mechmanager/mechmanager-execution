package service_test

import (
	"errors"
	"testing"

	service "mechmanager-execution/adapter/service"
	"mechmanager-execution/domain"

	"github.com/google/uuid"
)

func newAdapter() (*service.ExecutionAdapter, *mockRepo, *mockDynamo, *mockMessenger) {
	repo := newMockRepo()
	dynamo := &mockDynamo{}
	messenger := &mockMessenger{}
	adapter := service.NewExecutionAdapter(repo, dynamo, messenger)
	return adapter, repo, dynamo, messenger
}

// --- CreateFromOrder ---

func TestCreateFromOrder_Success(t *testing.T) {
	adapter, _, dynamo, _ := newAdapter()

	exec, err := adapter.CreateFromOrder("order-123", "mec-001")
	if err != nil {
		t.Fatalf("esperado sem erro, got: %v", err)
	}
	if exec.OrderID != "order-123" {
		t.Errorf("esperado order_id=order-123, got: %s", exec.OrderID)
	}
	if exec.Status != domain.ExecutionStatusQueued {
		t.Errorf("esperado status QUEUED, got: %s", exec.Status)
	}
	if len(dynamo.enqueued) != 1 || dynamo.enqueued[0] != "order-123" {
		t.Error("execução não foi enfileirada no DynamoDB")
	}
}

func TestCreateFromOrder_EmptyOrderID(t *testing.T) {
	adapter, _, _, _ := newAdapter()

	_, err := adapter.CreateFromOrder("", "mec-001")
	if err == nil {
		t.Fatal("esperado erro para order_id vazio")
	}
}

func TestCreateFromOrder_RepoError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	repo.saveErr = errors.New("db error")

	_, err := adapter.CreateFromOrder("order-123", "mec-001")
	if err == nil {
		t.Fatal("esperado erro quando repo falha")
	}
}

// --- UpdateStatus (Saga transitions) ---

func TestUpdateStatus_QueuedToInDiagnosis(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-001", "mec-001")

	updated, err := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "barulho no motor")
	if err != nil {
		t.Fatalf("transição QUEUED→IN_DIAGNOSIS falhou: %v", err)
	}
	if updated.Status != domain.ExecutionStatusInDiagnosis {
		t.Errorf("esperado IN_DIAGNOSIS, got: %s", updated.Status)
	}
	if updated.DiagnosticNotes != "barulho no motor" {
		t.Errorf("notas de diagnóstico não salvas")
	}
}

func TestUpdateStatus_InDiagnosisToInRepair(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-002", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "diagnóstico feito")

	updated, err := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "substituindo peça")
	if err != nil {
		t.Fatalf("transição IN_DIAGNOSIS→IN_REPAIR falhou: %v", err)
	}
	if updated.Status != domain.ExecutionStatusInRepair {
		t.Errorf("esperado IN_REPAIR, got: %s", updated.Status)
	}
}

func TestUpdateStatus_InvalidTransition(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-003", "mec-001")

	// QUEUED → IN_REPAIR é inválido (deve passar por IN_DIAGNOSIS)
	_, err := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")
	if err == nil {
		t.Fatal("esperado erro para transição inválida QUEUED→IN_REPAIR")
	}
}

func TestUpdateStatus_QueuedToCompleted_Invalid(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-004", "mec-001")

	_, err := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusCompleted, "")
	if err == nil {
		t.Fatal("esperado erro para transição inválida QUEUED→COMPLETED")
	}
}

func TestUpdateStatus_NotFound(t *testing.T) {
	adapter, _, _, _ := newAdapter()

	_, err := adapter.UpdateStatus(uuid.New(), domain.ExecutionStatusInDiagnosis, "")
	if err == nil {
		t.Fatal("esperado erro para ID inexistente")
	}
}

// --- Complete ---

func TestComplete_Success(t *testing.T) {
	adapter, _, dynamo, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-005", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	completed, err := adapter.Complete(exec.ID, "correia substituída")
	if err != nil {
		t.Fatalf("Complete falhou: %v", err)
	}
	if completed.Status != domain.ExecutionStatusCompleted {
		t.Errorf("esperado COMPLETED, got: %s", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("completed_at não foi preenchido")
	}
	if len(dynamo.removed) == 0 {
		t.Error("execução não foi removida do DynamoDB")
	}
	if len(messenger.completeEvents) == 0 {
		t.Error("evento EXECUTION_COMPLETED não foi enviado ao SQS")
	}
}

func TestComplete_NotInRepair(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-006", "mec-001")

	// Tentar concluir sem passar por IN_REPAIR
	_, err := adapter.Complete(exec.ID, "notas")
	if err == nil {
		t.Fatal("esperado erro ao concluir execução fora de IN_REPAIR")
	}
}

// --- Fail ---

func TestFail_FromQueued(t *testing.T) {
	adapter, _, _, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-007", "mec-001")

	failed, err := adapter.Fail(exec.ID, "peça indisponível")
	if err != nil {
		t.Fatalf("Fail falhou: %v", err)
	}
	if failed.Status != domain.ExecutionStatusFailed {
		t.Errorf("esperado FAILED, got: %s", failed.Status)
	}
	if len(messenger.failedEvents) == 0 {
		t.Error("evento EXECUTION_FAILED não foi enviado ao SQS (Saga rollback)")
	}
}

func TestFail_FromInDiagnosis(t *testing.T) {
	adapter, _, _, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-008", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "diagnóstico")

	failed, err := adapter.Fail(exec.ID, "falha durante diagnóstico")
	if err != nil {
		t.Fatalf("Fail de IN_DIAGNOSIS falhou: %v", err)
	}
	if failed.Status != domain.ExecutionStatusFailed {
		t.Errorf("esperado FAILED, got: %s", failed.Status)
	}
	if len(messenger.failedEvents) == 0 {
		t.Error("evento de compensação Saga não foi enviado")
	}
}

// --- ListAll ---

func TestListAll(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	adapter.CreateFromOrder("order-009", "mec-001")
	adapter.CreateFromOrder("order-010", "mec-002")

	all, err := adapter.ListAll()
	if err != nil {
		t.Fatalf("ListAll falhou: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("esperado 2 execuções, got: %d", len(all))
	}
}

// --- FindByOrderID ---

func TestFindByOrderID_Found(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	adapter.CreateFromOrder("order-011", "mec-001")

	exec, err := adapter.FindByOrderID("order-011")
	if err != nil {
		t.Fatalf("FindByOrderID falhou: %v", err)
	}
	if exec.OrderID != "order-011" {
		t.Errorf("esperado order_id=order-011, got: %s", exec.OrderID)
	}
}

func TestFindByOrderID_NotFound(t *testing.T) {
	adapter, _, _, _ := newAdapter()

	_, err := adapter.FindByOrderID("inexistente")
	if err == nil {
		t.Fatal("esperado erro para order_id inexistente")
	}
}

// --- FindByID ---

func TestFindByID_Found(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-012", "mec-001")

	found, err := adapter.FindByID(exec.ID)
	if err != nil {
		t.Fatalf("FindByID falhou: %v", err)
	}
	if found.ID != exec.ID {
		t.Errorf("ID incorreto: esperado %v, got %v", exec.ID, found.ID)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	adapter, _, _, _ := newAdapter()

	_, err := adapter.FindByID(uuid.New())
	if err == nil {
		t.Fatal("esperado erro para ID inexistente")
	}
}

// --- Complete error paths ---

func TestComplete_RepoUpdateError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-013", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	repo.updateErr = errors.New("db error")
	_, err := adapter.Complete(exec.ID, "notas")
	if err == nil {
		t.Fatal("esperado erro quando repo falha no Complete")
	}
}

func TestComplete_NotFound(t *testing.T) {
	adapter, _, _, _ := newAdapter()

	_, err := adapter.Complete(uuid.New(), "notas")
	if err == nil {
		t.Fatal("esperado erro para ID inexistente no Complete")
	}
}

// --- Fail error paths ---

func TestFail_NotFound(t *testing.T) {
	adapter, _, _, _ := newAdapter()

	_, err := adapter.Fail(uuid.New(), "motivo")
	if err == nil {
		t.Fatal("esperado erro para ID inexistente no Fail")
	}
}

func TestFail_RepoUpdateError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-014", "mec-001")

	repo.updateErr = errors.New("db error")
	_, err := adapter.Fail(exec.ID, "motivo")
	if err == nil {
		t.Fatal("esperado erro quando repo falha no Fail")
	}
}
