package controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

// StringToUint esta funcionalidad permite converitr los datos como son los params que hacen referencia a un id al formato de uint
func StringToUint(s string) (uint, error) {

	val, err := strconv.ParseUint(s, 10, 64) // Base 10 y 64 bits
	if err != nil {
		return 0, err
	}

	return uint(val), nil
}

// HandlerBindJson permite realizar el manejo de los datos que viene dentro del body y asignarlo a la variable data
func HandlerBindJson(c *gin.Context, data interface{}) bool {
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return true
	}
	return false
}

// HandlerInternalError permite realizar el manejo de errores para aquellas funciones que devuelven solo el err como las creaciones o actualizaciones
func HandlerInternalError(c *gin.Context, err error) bool {
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return true
	}
	return false
}

func HandlerCreatedSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func HandlerFound(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func HandlerFoundSuccess(c *gin.Context, err error, name string) bool {
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("%s not found", name)})
		return true
	}
	return false
}

func HandlerNotContent(c *gin.Context, err error) {
	c.JSON(http.StatusNoContent, gin.H{"error": err.Error()})
}
