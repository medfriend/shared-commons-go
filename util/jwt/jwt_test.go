package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/dgrijalva/jwt-go"
)

const testKey = "clave-de-prueba"

func firmar(t *testing.T, claims jwtlib.MapClaims) string {
	t.Helper()
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	token, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims).SignedString([]byte(testKey))
	if err != nil {
		t.Fatalf("firmar: %v", err)
	}
	return "Bearer " + token
}

// Un token de sesion como el que emite security-go: sin tipo, sin scopes.
func TestDecodeJWTSesionSinCamposNuevos(t *testing.T) {
	t.Setenv("JWT_KEY", testKey)

	claims, err := DecodeJWT(firmar(t, jwtlib.MapClaims{
		"user":      map[string]any{"usuario": 7, "usuario_id": 7},
		"entidadId": 3,
		"rolId":     2,
	}))
	if err != nil {
		t.Fatalf("DecodeJWT: %v", err)
	}
	if claims.User.UsuarioID != 7 || claims.EntidadId != 3 || claims.RolId != 2 {
		t.Fatalf("claims base mal parseados: %+v", claims)
	}
	if claims.IsApiKey() {
		t.Fatal("una sesion no es llave de API")
	}
	if claims.HasScope(ScopeQuery) {
		t.Fatal("una sesion no tiene scopes")
	}
}

// El JWT interno que firma el getway a partir de una llave de API.
func TestDecodeJWTLlaveDeApi(t *testing.T) {
	t.Setenv("JWT_KEY", testKey)

	claims, err := DecodeJWT(firmar(t, jwtlib.MapClaims{
		"user":      map[string]any{"usuario": 7, "usuario_id": 7},
		"entidadId": 3,
		"rolId":     2,
		"tipo":      TipoApiKey,
		"apiKeyId":  41,
		"scopes":    []string{ScopeQuery, ScopeInsert},
	}))
	if err != nil {
		t.Fatalf("DecodeJWT: %v", err)
	}
	if !claims.IsApiKey() || claims.ApiKeyId != 41 {
		t.Fatalf("no se reconocio la llave: %+v", claims)
	}
	if !claims.HasScope(ScopeQuery) || !claims.HasScope(ScopeInsert) {
		t.Fatal("scopes otorgados no se leen")
	}
	if claims.HasScope(ScopeUpdate) || claims.HasScope(ScopeDelete) {
		t.Fatal("scopes no otorgados no deben pasar")
	}
}

func TestDecodeJWTRechazaFirmaAjena(t *testing.T) {
	t.Setenv("JWT_KEY", "otra-clave")

	if _, err := DecodeJWT(firmar(t, jwtlib.MapClaims{"entidadId": 3})); err == nil {
		t.Fatal("token firmado con otra clave debe fallar")
	}
}

func TestHelpersConNil(t *testing.T) {
	var c *Claims
	if c.IsApiKey() || c.HasScope(ScopeQuery) {
		t.Fatal("claims nil deben responder false")
	}
}
