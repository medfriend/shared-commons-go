package consul

import (
	"encoding/json"
	"os"
)

// Secreto es un valor que no debe vivir en el KV de Consul: cualquiera que alcance el
// puerto 8500 lee el KV entero (JWT_KEY basta para firmar tokens de cualquier usuario).
// getway-go lo quita al publicar si CONSUL_SIN_SECRETOS=true y el servicio lo toma de su
// propia variable de entorno, con el mismo nombre que tiene en el .env del gateway.
type Secreto struct {
	// Key es la clave del KV (DB, RABBIT, MINIO...).
	Key string
	// Campo es el campo dentro del JSON de la clave. Vacio: la clave entera es el secreto
	// (JWT guarda la llave en crudo, no un JSON).
	Campo string
	// Env es la variable de entorno del servicio que trae el valor.
	Env string
}

// Secretos es la unica lista de secretos del KV: la usan el gateway para no publicarlos
// y GetKeyValue para completarlos desde el entorno. Un secreto nuevo es una linea aqui.
var Secretos = []Secreto{
	{Key: "JWT", Env: "JWT_KEY"},
	{Key: "DB", Campo: "DB_PASSWORD", Env: "DB_PASSWORD"},
	{Key: "DB_LOCAL", Campo: "DB_PASSWORD", Env: "DB_PASSWORD_LOCAL"},
	{Key: "REDIS", Campo: "REDIS_PASSWORD", Env: "REDIS_PASSWORD"},
	{Key: "REDIS_LOCAL", Campo: "REDIS_PASSWORD", Env: "REDIS_PASSWORD_LOCAL"},
	{Key: "CACHE", Campo: "REDIS_PASSWORD", Env: "REDIS_PASSWORD"},
	{Key: "RABBIT", Campo: "RABBIT_PASSWORD", Env: "RABBIT_PASSWORD"},
	{Key: "MINIO", Campo: "MINIO_ACCESS", Env: "MINIO_ACCESS"},
	{Key: "MINIO", Campo: "MINIO_SECRET", Env: "MINIO_SECRET"},
	{Key: "CHATBOT", Campo: "KEY", Env: "CHATBOT_KEY"},
	{Key: "WHATSAPP", Campo: "TOKEN", Env: "WHATSAPP_TOKEN"},
	{Key: "GMAILSENDER", Campo: "PASS", Env: "GMAILSENDER_PASS"},
}

func secretosDe(key string) []Secreto {
	var out []Secreto
	for _, s := range Secretos {
		if s.Key == key {
			out = append(out, s)
		}
	}
	return out
}

// QuitarSecretos devuelve el valor que el gateway publica para key sin sus secretos.
// publicar=false cuando la clave entera es secreta (JWT): no se escribe y se borra del
// KV. Un JSON que no se puede leer se devuelve tal cual: no se adivina su forma.
func QuitarSecretos(key, value string) (limpio string, publicar bool) {
	secretos := secretosDe(key)
	if len(secretos) == 0 {
		return value, true
	}
	var campos map[string]string
	for _, s := range secretos {
		if s.Campo == "" {
			return "", false
		}
		if campos == nil {
			if err := json.Unmarshal([]byte(value), &campos); err != nil {
				return value, true
			}
		}
		delete(campos, s.Campo)
	}
	out, err := json.Marshal(campos)
	if err != nil {
		return value, true
	}
	return string(out), true
}

// ConSecretos completa value (lo que vino del KV para key, "" si la clave no existe)
// con los secretos del entorno. La variable de entorno manda sobre el KV; si no esta
// definida queda lo que traiga el KV, asi un servicio sin las variables sigue
// funcionando mientras el gateway publique los secretos. faltan lista las variables
// que no estan en ninguno de los dos lados.
func ConSecretos(key, value string, getenv func(string) string) (completo string, faltan []string) {
	secretos := secretosDe(key)
	if len(secretos) == 0 {
		return value, nil
	}
	var campos map[string]string
	cambio := false
	for _, s := range secretos {
		env := getenv(s.Env)
		if s.Campo == "" {
			if env != "" {
				return env, nil
			}
			if value == "" {
				return value, []string{s.Env}
			}
			return value, nil
		}
		if campos == nil {
			if err := json.Unmarshal([]byte(value), &campos); err != nil || campos == nil {
				return value, nil
			}
		}
		if env != "" {
			if campos[s.Campo] != env {
				campos[s.Campo] = env
				cambio = true
			}
		} else if campos[s.Campo] == "" {
			faltan = append(faltan, s.Env)
		}
	}
	if !cambio {
		return value, faltan
	}
	out, err := json.Marshal(campos)
	if err != nil {
		return value, faltan
	}
	return string(out), faltan
}

func conSecretosDelEntorno(key, value string) (string, []string) {
	return ConSecretos(key, value, os.Getenv)
}
