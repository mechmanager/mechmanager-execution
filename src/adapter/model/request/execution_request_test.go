package request_test

import (
	"encoding/json"
	"testing"

	"mechmanager-execution/adapter/model/request"
)

func TestUpdateExecutionStatusInput(t *testing.T) {
	raw := `{"status":"IN_DIAGNOSIS","mechanic_id":"mec-1","diagnostic_notes":"barulho"}`
	var r request.UpdateExecutionStatusInput
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.Status != "IN_DIAGNOSIS" {
		t.Errorf("esperado IN_DIAGNOSIS, got %s", r.Status)
	}
}

func TestCompleteExecutionInput(t *testing.T) {
	raw := `{"repair_notes":"correia trocada"}`
	var r request.CompleteExecutionInput
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.RepairNotes != "correia trocada" {
		t.Errorf("esperado 'correia trocada', got %s", r.RepairNotes)
	}
}

func TestFailExecutionInput(t *testing.T) {
	raw := `{"reason":"peça indisponível"}`
	var r request.FailExecutionInput
	if err := json.Unmarshal([]byte(raw), &r); err != nil {
		t.Fatal(err)
	}
	if r.Reason != "peça indisponível" {
		t.Errorf("esperado 'peça indisponível', got %s", r.Reason)
	}
}
