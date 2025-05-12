package jwt

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"os"
	"strings"
)

type Claims struct {
	User struct {
		Usuario   int `json:"usuario"`
		UsuarioID int `json:"usuario_id"`
	} `json:"user"`
	EntidadId string `json:"entidadId"`
	jwt.StandardClaims
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
		fmt.Println("token invalido")
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
