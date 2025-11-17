package factory

import (
	"fmt"
	"github.com/gin-gonic/gin"
	ctrlutil "github.com/medfriend/shared-commons-go/util/controller"
)

type FactoryController struct {
	svc FactoryService
}

func NewFactoryController(svc FactoryService) *FactoryController {
	return &FactoryController{svc: svc}
}

// CreateTable @Summary Crea una nueva tabla dinámica de patrón
// @Tags factory
// @Accept json
// @Produce json
// @Param payload body CreateTableDTO true "Datos de la tabla"
// @Success 201 {object} map[string]string
// @Router /factory/CreateTable [post]
func (fc *FactoryController) CreateTable(c *gin.Context) {
	var payload CreateTableDTO
	if ctrlutil.HandlerBindJson(c, &payload) {
		return
	}
	if ctrlutil.HandlerInternalError(c, fc.svc.CreateTable(payload)) {
		return
	}
	ctrlutil.HandlerCreatedSuccess(c, map[string]string{"status": "created", "table": payload.Name}, 0)
}

// Query @Summary Lee filas con filtros estilo Directus
// @Tags factory
// @Accept json
// @Produce json
// @Param table path string true "Nombre de la tabla"
// @Param payload body QueryDTO true "Consulta"
// @Success 200 {object} dto.PaginatedResponse
// @Router /factory/Query/{table} [post]
func (fc *FactoryController) Query(c *gin.Context) {
	table := c.Param("table")
	var q QueryDTO
	if ctrlutil.HandlerBindJson(c, &q) {
		return
	}
	res, err := fc.svc.Query(table, q)
	if ctrlutil.HandlerFoundSuccess(c, err, "rows") {
		return
	}
	ctrlutil.HandlerCreatedSuccess(c, res, 0)
}

// Update @Summary Actualiza filas con filtros estilo Directus
// @Tags factory
// @Accept json
// @Produce json
// @Param table path string true "Nombre de la tabla"
// @Param payload body UpdateDTO true "Actualización"
// @Success 200 {object} map[string]interface{}
// @Router /factory/Update/{table} [patch]
func (fc *FactoryController) Update(c *gin.Context) {
    table := c.Param("table")
    fmt.Println("[Controller][Update] table:", table)
    var upd UpdateDTO
    if ctrlutil.HandlerBindJson(c, &upd) {
        fmt.Println("[Controller][Update] bind error for table:", table)
        return
    }
    fmt.Println("[Controller][Update] payload:", upd)
    rows, err := fc.svc.Update(table, upd)
    fmt.Println("[Controller][Update] service response rows:", rows, "err:", err)
    if ctrlutil.HandlerInternalError(c, err) {
        return
    }
    ctrlutil.HandlerCreatedSuccess(c, map[string]interface{}{"updated": rows}, 0)
}

// DropTable @Summary Elimina una tabla completa
// @Tags factory
// @Produce json
// @Param table path string true "Nombre de la tabla"
// @Success 204
// @Router /factory/DropTable/{table} [delete]
func (fc *FactoryController) DropTable(c *gin.Context) {
	table := c.Param("table")
	if ctrlutil.HandlerInternalError(c, fc.svc.DropTable(table)) {
		return
	}
	ctrlutil.HandlerDeleteSuccess(c, map[string]string{"table": table}, 0)
}

// Relation @Summary Crea una relación FK y el campo en la segunda tabla
// @Tags factory
// @Accept json
// @Produce json
// @Param payload body RelationDTO true "Relación"
// @Success 201 {object} map[string]string
// @Router /factory/Relation [post]
func (fc *FactoryController) Relation(c *gin.Context) {
	var payload RelationDTO
	if ctrlutil.HandlerBindJson(c, &payload) {
		return
	}
	if ctrlutil.HandlerInternalError(c, fc.svc.CreateRelation(payload)) {
		return
	}
	ctrlutil.HandlerCreatedSuccess(c, map[string]string{"status": "relation_created"}, 0)
}

// Insert @Summary Inserta una fila en una tabla dinámica
// @Tags factory
// @Accept json
// @Produce json
// @Param table path string true "Nombre de la tabla"
// @Param payload body InsertDTO true "Datos"
// @Success 201 {object} map[string]interface{}
// @Router /factory/Insert/{table} [post]
func (fc *FactoryController) Insert(c *gin.Context) {
	table := c.Param("table")
	var payload InsertDTO
	if ctrlutil.HandlerBindJson(c, &payload) {
		return
	}
	id, err := fc.svc.Insert(table, payload)
	if ctrlutil.HandlerInternalError(c, err) {
		return
	}
	ctrlutil.HandlerCreatedSuccess(c, map[string]interface{}{"id": id}, uint(id))
}

// Delete @Summary Elimina filas por filtros (requiere al menos un filtro)
// @Tags factory
// @Accept json
// @Produce json
// @Param table path string true "Nombre de la tabla"
// @Param payload body DeleteDTO true "Filtros"
// @Success 200 {object} map[string]interface{}
// @Router /factory/Delete/{table} [delete]
func (fc *FactoryController) Delete(c *gin.Context) {
	table := c.Param("table")
	var payload DeleteDTO
	if ctrlutil.HandlerBindJson(c, &payload) {
		return
	}
	rows, err := fc.svc.Delete(table, payload)
	if ctrlutil.HandlerInternalError(c, err) {
		return
	}
	ctrlutil.HandlerCreatedSuccess(c, map[string]interface{}{"deleted": rows}, 0)
}
