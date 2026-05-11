package controllers

import (
	"net/http"

	"mechmanager-execution/adapter/model/request"
	"mechmanager-execution/adapter/model/response"
	port "mechmanager-execution/application/port/input"
	"mechmanager-execution/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ExecutionController struct {
	useCase port.ExecutionUseCaseInterface
}

func NewExecutionController(uc port.ExecutionUseCaseInterface) *ExecutionController {
	return &ExecutionController{useCase: uc}
}

// ListAll godoc
// @Summary Lista todas as execuções
// @Description Retorna a lista de todas as execuções de OS
// @Tags Execution
// @Produce json
// @Success 200 {array} response.ExecutionResponse
// @Router /executions [get]
func (ctrl *ExecutionController) ListAll(c *gin.Context) {
	executions, err := ctrl.useCase.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]*response.ExecutionResponse, 0, len(executions))
	for _, e := range executions {
		result = append(result, response.FromDomain(e))
	}
	c.JSON(http.StatusOK, result)
}

// FindByID godoc
// @Summary Busca execução por ID
// @Tags Execution
// @Produce json
// @Param id path string true "ID da execução"
// @Success 200 {object} response.ExecutionResponse
// @Router /executions/{id} [get]
func (ctrl *ExecutionController) FindByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	execution, err := ctrl.useCase.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execução não encontrada"})
		return
	}
	c.JSON(http.StatusOK, response.FromDomain(execution))
}

// FindByOrderID godoc
// @Summary Busca execução por Order ID
// @Tags Execution
// @Produce json
// @Param id path string true "ID da ordem de serviço"
// @Success 200 {object} response.ExecutionResponse
// @Router /executions/order/{id} [get]
func (ctrl *ExecutionController) FindByOrderID(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id inválido"})
		return
	}
	execution, err := ctrl.useCase.FindByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Execução não encontrada para esta OS"})
		return
	}
	c.JSON(http.StatusOK, response.FromDomain(execution))
}

// UpdateStatus godoc
// @Summary Atualiza status da execução
// @Description Atualiza o status: QUEUED → IN_DIAGNOSIS → IN_REPAIR
// @Tags Execution
// @Accept json
// @Produce json
// @Param id path string true "ID da execução"
// @Param input body request.UpdateExecutionStatusInput true "Dados de atualização"
// @Success 200 {object} response.ExecutionResponse
// @Router /executions/{id}/status [patch]
func (ctrl *ExecutionController) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var input request.UpdateExecutionStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	notes := input.DiagnosticNotes
	if input.RepairNotes != "" {
		notes = input.RepairNotes
	}
	updated, err := ctrl.useCase.UpdateStatus(id, domain.ExecutionStatus(input.Status), notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response.FromDomain(updated))
}

// Complete godoc
// @Summary Conclui a execução da OS
// @Description Marca a execução como COMPLETED e notifica o OS Service via SQS
// @Tags Execution
// @Accept json
// @Produce json
// @Param id path string true "ID da execução"
// @Param input body request.CompleteExecutionInput true "Notas de reparo"
// @Success 200 {object} response.ExecutionResponse
// @Router /executions/{id}/complete [patch]
func (ctrl *ExecutionController) Complete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var input request.CompleteExecutionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	completed, err := ctrl.useCase.Complete(id, input.RepairNotes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response.FromDomain(completed))
}

// Fail godoc
// @Summary Registra falha na execução (Saga rollback)
// @Description Marca como FAILED e envia evento de compensação ao OS Service via SQS
// @Tags Execution
// @Accept json
// @Produce json
// @Param id path string true "ID da execução"
// @Param input body request.FailExecutionInput true "Motivo da falha"
// @Success 200 {object} response.ExecutionResponse
// @Router /executions/{id}/fail [patch]
func (ctrl *ExecutionController) Fail(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID inválido"})
		return
	}
	var input request.FailExecutionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	failed, err := ctrl.useCase.Fail(id, input.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, response.FromDomain(failed))
}
