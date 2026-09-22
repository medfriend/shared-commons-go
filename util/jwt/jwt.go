package jwt

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/medfriend/shared-commons-go/util/logger"
	"go.uber.org/zap"
	"os"
	"strings"
)

// Tipos de token que firma la plataforma. Un token de sesion no trae "tipo".
const (
	// TipoApiKey marca el JWT interno que el getway firma cuando una peticion
	// entra con X-Api-Key. Los servicios lo usan para acotar el factory.
	TipoApiKey = "api_key"
	// TipoRecuperarClave marca el JWT del flujo "olvide mi contrasena"; el getway
	// lo rechaza como sesion y solo vale en las rutas publicas de recuperacion.
	TipoRecuperarClave = "recuperar_clave"
)

// Scopes de una llave de API. Cada uno habilita una operacion del factory.
const (
	ScopeQuery  = "query"  // factory/Query
	ScopeInsert = "insert" // factory/Insert
	ScopeUpdate = "update" // factory/Update
	ScopeDelete = "delete" // factory/Delete
)

// Claims son los datos que la plataforma mete en el JWT. Todo campo nuevo va con
// omitempty: los tokens de sesion que emite security-go no lo traen y deben
// seguir parseando igual.
type Claims struct {
	User struct {
		Usuario   int `json:"usuario"`
		UsuarioID int `json:"usuario_id"`
	} `json:"user"`
	EntidadId int `json:"entidadId"`
	RolId     int `json:"rolId,omitempty"`
	// Tipo distingue el origen del token: "" sesion, TipoApiKey, TipoRecuperarClave.
	Tipo string `json:"tipo,omitempty"`
	// ApiKeyId y Scopes solo vienen cuando Tipo == TipoApiKey.
	ApiKeyId int      `json:"apiKeyId,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	jwt.StandardClaims
}

// IsApiKey reporta si el token nacio de una llave de API y no de un login.
func (c *Claims) IsApiKey() bool {
	return c != nil && c.Tipo == TipoApiKey
}

// HasScope reporta si la llave tiene habilitada la operacion. Un token de sesion
// no tiene scopes: para el responde false siempre, asi que quien lo llame debe
// preguntar IsApiKey primero y no usarlo como puerta de una sesion normal.
func (c *Claims) HasScope(scope string) bool {
	if c == nil {
		return false
	}
	for _, s := range c.Scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func ValidateJWT(tokenString string) (*jwt.Token, error) {

	jwtKey := []byte(os.Getenv("JWT_KEY"))

	tokenSplited, err := splitToken(tokenString)

	if err != nil {
		return nil, err
	}

	token, err := jwt.Parse(tokenSplited, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func splitToken(tokenString string) (string, error) {
	parts := strings.Split(tokenString, " ")

	if len(parts) != 2 || parts[0] != "Bearer" {
		logger.GetLogger().Warn("Token inválido recibido", zap.String("token", tokenString))
		return "", errors.New("token invalido")
	}

	return parts[1], nil
}

func DecodeJWT(tokenString string) (*Claims, error) {
	jwtKey := []byte(os.Getenv("JWT_KEY"))

	// Dividir el token para extraer solo el token JWT
	tokenSplited, err := splitToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Parsear y decodificar el token con los claims
	token, err := jwt.ParseWithClaims(tokenSplited, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}

	// Verificar y extraer los claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	} else {
		return nil, errors.New("token inválido o claims no válidos")
	}
}
