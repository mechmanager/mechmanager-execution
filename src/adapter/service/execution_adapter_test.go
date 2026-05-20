package service_test

import (
	"errors"
	"testing"
	"time"

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

func TestCreateFromOrder_DynamoError(t *testing.T) {
	adapter, _, dynamo, _ := newAdapter()
	dynamo.enqueueErr = errors.New("dynamo error")

	exec, err := adapter.CreateFromOrder("order-012", "mec-001")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if exec.OrderID != "order-012" {
		t.Errorf("esperado order_id=order-012, got: %s", exec.OrderID)
	}
}

func TestComplete_RepoUpdateError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-015", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	repo.updateErr = errors.New("update error")
	_, err := adapter.Complete(exec.ID, "finalizando")
	if err == nil {
		t.Fatal("esperado erro quando repo.Update falha em Complete")
	}
}

func TestComplete_DynamoError(t *testing.T) {
	adapter, _, dynamo, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-016", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	dynamo.removeErr = errors.New("remove error")
	completed, err := adapter.Complete(exec.ID, "finalizado")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if completed.Status != domain.ExecutionStatusCompleted {
		t.Errorf("esperado COMPLETED, got: %s", completed.Status)
	}
}

func TestFail_RepoUpdateError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-018", "mec-001")

	repo.updateErr = errors.New("update error")
	_, err := adapter.Fail(exec.ID, "falha")
	if err == nil {
		t.Fatal("esperado erro quando repo.Update falha em Fail")
	}
}

func TestIsValidTransition(t *testing.T) {
	if !service.IsValidTransition(domain.ExecutionStatusQueued, domain.ExecutionStatusInDiagnosis) {
		t.Error("esperado transição válida QUEUED→IN_DIAGNOSIS")
	}
	if service.IsValidTransition(domain.ExecutionStatusQueued, domain.ExecutionStatusCompleted) {
		t.Error("esperado transição inválida QUEUED→COMPLETED")
	}
}

func TestUpdateStatus_RepoUpdateError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-021", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")

	repo.updateErr = errors.New("update error")
	_, err := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")
	if err == nil {
		t.Fatal("esperado erro quando repo.Update falha em UpdateStatus")
	}
}

func TestUpdateStatus_DynamoError(t *testing.T) {
	adapter, _, dynamo, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-022", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")

	dynamo.updateErr = errors.New("dynamo error")
	updated, err := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "reparo")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if updated.Status != domain.ExecutionStatusInRepair {
		t.Errorf("esperado IN_REPAIR, got: %s", updated.Status)
	}
}

func TestComplete_MessengerError(t *testing.T) {
	adapter, _, _, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-023", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	messenger.completeErr = errors.New("messenger error")
	completed, err := adapter.Complete(exec.ID, "finalizado")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if completed.Status != domain.ExecutionStatusCompleted {
		t.Errorf("esperado COMPLETED, got: %s", completed.Status)
	}
}

func TestFail_DynamoError(t *testing.T) {
	adapter, _, dynamo, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-024", "mec-001")

	dynamo.updateErr = errors.New("dynamo error")
	failed, err := adapter.Fail(exec.ID, "falha")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if failed.Status != domain.ExecutionStatusFailed {
		t.Errorf("esperado FAILED, got: %s", failed.Status)
	}
}

func TestFail_MessengerError(t *testing.T) {
	adapter, _, _, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-025", "mec-001")

	messenger.failedErr = errors.New("messenger error")
	failed, err := adapter.Fail(exec.ID, "falha")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if failed.Status != domain.ExecutionStatusFailed {
		t.Errorf("esperado FAILED, got: %s", failed.Status)
	}
}

func TestFindByID_NotFound(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	_, err := adapter.FindByID(uuid.New())
	if err == nil {
		t.Fatal("esperado erro para ID inexistente em FindByID")
	}
}

func TestFail_SavesDiagnosticNotes(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-030", "mec-001")

	failed, _ := adapter.Fail(exec.ID, "motor queimado")
	if failed.DiagnosticNotes != "motor queimado" {
		t.Errorf("esperado DiagnosticNotes='motor queimado', got: %s", failed.DiagnosticNotes)
	}
}

func TestUpdateStatus_SavesRepairNotes(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-031", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "diagnóstico feito")

	updated, _ := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "troca de correia")
	if updated.RepairNotes != "troca de correia" {
		t.Errorf("esperado RepairNotes='troca de correia', got: %s", updated.RepairNotes)
	}
}

func TestComplete_SavesRepairNotes(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-032", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	completed, _ := adapter.Complete(exec.ID, "troca de embreagem")
	if completed.RepairNotes != "troca de embreagem" {
		t.Errorf("esperado RepairNotes='troca de embreagem', got: %s", completed.RepairNotes)
	}
}

func TestFindByID_Found(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-033", "mec-001")

	found, err := adapter.FindByID(exec.ID)
	if err != nil {
		t.Fatalf("FindByID falhou: %v", err)
	}
	if found.ID != exec.ID {
		t.Errorf("esperado ID=%s, got: %s", exec.ID, found.ID)
	}
}

func TestIsValidTransition_UnknownStatus(t *testing.T) {
	unknown := domain.ExecutionStatus("UNKNOWN")
	if service.IsValidTransition(unknown, domain.ExecutionStatusQueued) {
		t.Error("esperado false para status desconhecido")
	}
}

func TestFail_UpdatesTimestamp(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-034", "mec-001")

	before := exec.UpdatedAt
	// força o relógio avançar
	time.Sleep(2 * time.Millisecond)

	failed, _ := adapter.Fail(exec.ID, "falha de teste")
	if !failed.UpdatedAt.After(before) {
		t.Errorf("esperado UpdatedAt atualizado em Fail, before=%v after=%v", before, failed.UpdatedAt)
	}
}

func TestUpdateStatus_UpdatesTimestamp(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-035", "mec-001")

	before := exec.UpdatedAt
	// força o relógio avançar
	time.Sleep(2 * time.Millisecond)

	updated, _ := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "teste")
	if !updated.UpdatedAt.After(before) {
		t.Errorf("esperado UpdatedAt atualizado em UpdateStatus, before=%v after=%v", before, updated.UpdatedAt)
	}
}

func TestCreateFromOrder_SavesMechanicID(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-040", "mec-999")

	if exec.MechanicID != "mec-999" {
		t.Errorf("esperado MechanicID='mec-999', got: %s", exec.MechanicID)
	}
}

func TestComplete_UpdatesTimestamp(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-041", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	before := exec.UpdatedAt
	time.Sleep(2 * time.Millisecond)

	completed, _ := adapter.Complete(exec.ID, "finalizado")
	if !completed.UpdatedAt.After(before) {
		t.Errorf("esperado UpdatedAt atualizado em Complete, before=%v after=%v", before, completed.UpdatedAt)
	}
}

func TestFail_FromInRepair(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-042", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	failed, _ := adapter.Fail(exec.ID, "falha durante reparo")
	if failed.Status != domain.ExecutionStatusFailed {
		t.Errorf("esperado FAILED, got: %s", failed.Status)
	}
}

func TestUpdateStatus_SavesDiagnosticNotes(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-043", "mec-001")

	updated, _ := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "teste diagnóstico")
	if updated.DiagnosticNotes != "teste diagnóstico" {
		t.Errorf("esperado DiagnosticNotes='teste diagnóstico', got: %s", updated.DiagnosticNotes)
	}
}

func TestIsValidTransition_AllValidPaths(t *testing.T) {
	valid := []struct {
		current domain.ExecutionStatus
		next    domain.ExecutionStatus
	}{
		{domain.ExecutionStatusQueued, domain.ExecutionStatusInDiagnosis},
		{domain.ExecutionStatusQueued, domain.ExecutionStatusFailed},
		{domain.ExecutionStatusInDiagnosis, domain.ExecutionStatusInRepair},
		{domain.ExecutionStatusInDiagnosis, domain.ExecutionStatusFailed},
		{domain.ExecutionStatusInRepair, domain.ExecutionStatusCompleted},
		{domain.ExecutionStatusInRepair, domain.ExecutionStatusFailed},
	}
	for _, v := range valid {
		if !service.IsValidTransition(v.current, v.next) {
			t.Errorf("esperado transição válida %s→%s", v.current, v.next)
		}
	}
}

func TestCreateFromOrder_RepoError_NoEnqueue(t *testing.T) {
	adapter, repo, dynamo, _ := newAdapter()
	repo.saveErr = errors.New("db error")

	_, err := adapter.CreateFromOrder("order-044", "mec-001")
	if err == nil {
		t.Fatal("esperado erro quando repo.Save falha")
	}
	if len(dynamo.enqueued) != 0 {
		t.Error("não deveria enfileirar quando repo.Save falha")
	}
}

func TestUpdateStatus_FindByIDError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	repo.findErr = errors.New("db error")

	_, err := adapter.UpdateStatus(uuid.New(), domain.ExecutionStatusInDiagnosis, "")
	if err == nil {
		t.Fatal("esperado erro quando repo.FindByID falha")
	}
}

func TestComplete_FindByIDError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	repo.findErr = errors.New("db error")

	_, err := adapter.Complete(uuid.New(), "notas")
	if err == nil {
		t.Fatal("esperado erro quando repo.FindByID falha em Complete")
	}
}

func TestFail_FindByIDError(t *testing.T) {
	adapter, repo, _, _ := newAdapter()
	repo.findErr = errors.New("db error")

	_, err := adapter.Fail(uuid.New(), "falha")
	if err == nil {
		t.Fatal("esperado erro quando repo.FindByID falha em Fail")
	}
}

func TestIsValidTransition_InvalidWithinMap(t *testing.T) {
	if service.IsValidTransition(domain.ExecutionStatusInDiagnosis, domain.ExecutionStatusCompleted) {
		t.Error("esperado transição inválida IN_DIAGNOSIS→COMPLETED")
	}
}

func TestComplete_MessengerSendError(t *testing.T) {
	adapter, _, _, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-045", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "")

	messenger.completeErr = errors.New("send error")
	completed, err := adapter.Complete(exec.ID, "finalizado")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if completed.Status != domain.ExecutionStatusCompleted {
		t.Errorf("esperado COMPLETED, got: %s", completed.Status)
	}
}

func TestFail_MessengerSendError(t *testing.T) {
	adapter, _, _, messenger := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-046", "mec-001")

	messenger.failedErr = errors.New("send error")
	failed, err := adapter.Fail(exec.ID, "falha simulada")
	if err != nil {
		t.Fatalf("não deveria falhar, got: %v", err)
	}
	if failed.Status != domain.ExecutionStatusFailed {
		t.Errorf("esperado FAILED, got: %s", failed.Status)
	}
}

func TestUpdateStatus_FillsRepairNotes(t *testing.T) {
	adapter, _, _, _ := newAdapter()
	exec, _ := adapter.CreateFromOrder("order-047", "mec-001")
	exec, _ = adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInDiagnosis, "diagnóstico feito")

	updated, _ := adapter.UpdateStatus(exec.ID, domain.ExecutionStatusInRepair, "troca de motor")
	if updated.RepairNotes != "troca de motor" {
		t.Errorf("esperado RepairNotes='troca de motor', got: %s", updated.RepairNotes)
	}
}
