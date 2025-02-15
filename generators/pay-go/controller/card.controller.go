package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/medfriend/shared-commons-go/util/controller"
	"pay-go/service"
)

type CardController struct {
	CardService service.CardService
}

func NewCardController(CardService service.CardService) *CardController {
	return &CardController{
		CardService: CardService,
	}
}

// CreateCard crea una nueva entidad
// @Summary Crear una entidad
// @Security      BearerAuth
// @Description Este endpoint permite crear una nueva entidad en el sistema.
// @Tags entidades
// @Accept json
// @Produce json
// @Param Card body entity.Card true "Información de la entidad"
// @Success 201 {object} entity.Card "Card creada con éxito"
// @Failure 400 {object} map[string]string "Error en el cuerpo de la solicitud"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /Card [post]
func (ctrl *CardController) CreateCard(c *gin.Context) {
	var Card entity.Card

	controller.HandlerBindJson(c, &Card)
	controller.HandlerInternalError(c, ctrl.CardService.CreateCard(&Card))
	controller.HandlerCreatedSuccess(c, Card)
}

// GetCardById obtiene una entidad por su ID
// @Summary Obtener una entidad por ID
// @Security      BearerAuth
// @Description Este endpoint permite obtener la información de una entidad específica usando su ID.
// @Tags entidades
// @Accept json
// @Produce json
// @Param id path uint true "ID de la entidad"
// @Success 200 {object} entity.Card "Card encontrada"
// @Failure 404 {object} map[string]string "Card no encontrada"
// @Router /Card/{id} [get]
func (ctrl *CardController) GetCardById(c *gin.Context) {
	id, err := controller.StringToUint(c.Param("id"))
	Card, err := ctrl.CardService.GetCardById(id)
	controller.HandlerFoundSuccess(c, err, "entidad")
	controller.HandlerCreatedSuccess(c, Card)
}

// GetAllCards obtiene todas las entidades registradas
// @Summary Obtener todas las entidades
// @Security      BearerAuth
// @Description Este endpoint obtener todas las entidades
// @Tags entidades
// @Accept json
// @Produce json
// @Success 200 {array} entity.Card "Cards encontradas"
// @Failure 404 {object} map[string]string "Cards no encontradas"
// @Router /Card/all [get]
func (ctrl *CardController) GetAllCards(c *gin.Context) {
	Card, err := ctrl.CardService.GetAllCards()
	controller.HandlerFoundSuccess(c, err, "entidad")
	controller.HandlerCreatedSuccess(c, Card)
}

// UpdateCard actualiza una entidad existente
// @Summary Actualizar una entidad
// @Security      BearerAuth
// @Description Este endpoint permite actualizar la información de una entidad existente.
// @Tags entidades
// @Accept json
// @Produce json
// @Param Card body entity.Card true "Información de la entidad actualizada"
// @Success 200 {object} entity.Card "Card actualizada con éxito"
// @Failure 400 {object} map[string]string "Error en el cuerpo de la solicitud"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /Card [put]
func (ctrl *CardController) UpdateCard(c *gin.Context) {
	var Card entity.Card
	controller.HandlerBindJson(c, &Card)
	controller.HandlerInternalError(c, ctrl.CardService.UpdateCard(&Card))
	controller.HandlerCreatedSuccess(c, Card)
}

// DeleteCard elimina una entidad por su ID
// @Summary Eliminar una entidad
// @Security      BearerAuth
// @Description Este endpoint permite eliminar una entidad específica usando su ID.
// @Tags entidades
// @Accept json
// @Produce json
// @Param id path uint true "ID de la entidad"
// @Success 204 "Card eliminada con éxito"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /Card/{id} [delete]
func (ctrl *CardController) DeleteCard(c *gin.Context) {
	id, _ := controller.StringToUint(c.Param("id"))
	controller.HandlerInternalError(c, ctrl.CardService.DeleteCard(id))
	controller.HandlerNotContent(c, nil)
}


