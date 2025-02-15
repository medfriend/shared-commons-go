package template

import (
	"fmt"
	"github.com/medfriend/shared-commons-go/generators/util"
)

func GetController(args []string) string {
	capitalized := util.CapitalizeFirst(args[0])
	return fmt.Sprintf(`package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/medfriend/shared-commons-go/util/controller"
	"%s-go/service"
    "%s-go/entity"
)

type %sController struct {
	%sService service.%sService
}

func New%sController(%sService service.%sService) *%sController {
	return &%sController{
		%sService: %sService,
	}
}

// Create%s crea una nueva entidad
// @Summary Crear una entidad
// @Security      BearerAuth
// @Description Este endpoint permite crear una nueva entidad en el sistema.
// @Tags entidades
// @Accept json
// @Produce json
// @Param %s body entity.%s true "Información de la entidad"
// @Success 201 {object} entity.%s "%s creada con éxito"
// @Failure 400 {object} map[string]string "Error en el cuerpo de la solicitud"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /%s [post]
func (ctrl *%sController) Create%s(c *gin.Context) {
	var %s entity.%s

	controller.HandlerBindJson(c, &%s)
	controller.HandlerInternalError(c, ctrl.%sService.Create%s(&%s))
	controller.HandlerCreatedSuccess(c, %s)
}

// Get%sById obtiene una entidad por su ID
// @Summary Obtener una entidad por ID
// @Security      BearerAuth
// @Description Este endpoint permite obtener la información de una entidad específica usando su ID.
// @Tags entidades
// @Accept json
// @Produce json
// @Param id path uint true "ID de la entidad"
// @Success 200 {object} entity.%s "%s encontrada"
// @Failure 404 {object} map[string]string "%s no encontrada"
// @Router /%s/{id} [get]
func (ctrl *%sController) Get%sById(c *gin.Context) {
	id, err := controller.StringToUint(c.Param("id"))
	%s, err := ctrl.%sService.Get%sById(id)
	controller.HandlerFoundSuccess(c, err, "entidad")
	controller.HandlerCreatedSuccess(c, %s)
}

// GetAll%ss obtiene todas las entidades registradas
// @Summary Obtener todas las entidades
// @Security      BearerAuth
// @Description Este endpoint obtener todas las entidades
// @Tags entidades
// @Accept json
// @Produce json
// @Success 200 {array} entity.%s "%ss encontradas"
// @Failure 404 {object} map[string]string "%ss no encontradas"
// @Router /%s/all [get]
func (ctrl *%sController) GetAll%ss(c *gin.Context) {
	%s, err := ctrl.%sService.GetAll%ss()
	controller.HandlerFoundSuccess(c, err, "entidad")
	controller.HandlerCreatedSuccess(c, %s)
}

// Update%s actualiza una entidad existente
// @Summary Actualizar una entidad
// @Security      BearerAuth
// @Description Este endpoint permite actualizar la información de una entidad existente.
// @Tags entidades
// @Accept json
// @Produce json
// @Param %s body entity.%s true "Información de la entidad actualizada"
// @Success 200 {object} entity.%s "%s actualizada con éxito"
// @Failure 400 {object} map[string]string "Error en el cuerpo de la solicitud"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /%s [put]
func (ctrl *%sController) Update%s(c *gin.Context) {
	var %s entity.%s
	controller.HandlerBindJson(c, &%s)
	controller.HandlerInternalError(c, ctrl.%sService.Update%s(&%s))
	controller.HandlerCreatedSuccess(c, %s)
}

// Delete%s elimina una entidad por su ID
// @Summary Eliminar una entidad
// @Security      BearerAuth
// @Description Este endpoint permite eliminar una entidad específica usando su ID.
// @Tags entidades
// @Accept json
// @Produce json
// @Param id path uint true "ID de la entidad"
// @Success 204 "%s eliminada con éxito"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /%s/{id} [delete]
func (ctrl *%sController) Delete%s(c *gin.Context) {
	id, _ := controller.StringToUint(c.Param("id"))
	controller.HandlerInternalError(c, ctrl.%sService.Delete%s(id))
	controller.HandlerNotContent(c, nil)
}


`, args[1], args[1], capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized,
		capitalized, capitalized, capitalized, capitalized, capitalized)
}
