package consul

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func entorno(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func campos(t *testing.T, raw string) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("json %q: %v", raw, err)
	}
	return m
}

func TestQuitarSecretosDejaLaDireccion(t *testing.T) {
	limpio, publicar := QuitarSecretos("DB", `{"DB_HOST":"db","DB_USER":"pai","DB_PASSWORD":"x"}`)
	if !publicar {
		t.Fatal("DB se sigue publicando sin la contrasena")
	}
	want := map[string]string{"DB_HOST": "db", "DB_USER": "pai"}
	if got := campos(t, limpio); !reflect.DeepEqual(got, want) {
		t.Fatalf("DB = %v, quiero %v", got, want)
	}
}

func TestQuitarSecretosMinioQuitaLosDos(t *testing.T) {
	limpio, _ := QuitarSecretos("MINIO", `{"MINIO_HOST":"m","MINIO_ACCESS":"a","MINIO_SECRET":"s"}`)
	if got := campos(t, limpio); !reflect.DeepEqual(got, map[string]string{"MINIO_HOST": "m"}) {
		t.Fatalf("MINIO = %v", got)
	}
}

func TestQuitarSecretosJWTNoSePublica(t *testing.T) {
	if _, publicar := QuitarSecretos("JWT", "llave"); publicar {
		t.Fatal("JWT es la llave entera: no se publica")
	}
}

func TestQuitarSecretosClaveSinSecretosIntacta(t *testing.T) {
	raw := `{"SERVICE_PORT":"9060"}`
	if limpio, publicar := QuitarSecretos("PAI360", raw); limpio != raw || !publicar {
		t.Fatalf("PAI360 = %q %v", limpio, publicar)
	}
}

// El gateway publica sin secretos y el servicio los toma de su entorno.
func TestConSecretosCompletaDesdeElEntorno(t *testing.T) {
	limpio, _ := QuitarSecretos("RABBIT", `{"RABBIT_HOST":"r","RABBIT_PASSWORD":"vieja"}`)
	completo, faltan := ConSecretos("RABBIT", limpio, entorno(map[string]string{"RABBIT_PASSWORD": "nueva"}))
	if len(faltan) != 0 {
		t.Fatalf("faltan = %v", faltan)
	}
	want := map[string]string{"RABBIT_HOST": "r", "RABBIT_PASSWORD": "nueva"}
	if got := campos(t, completo); !reflect.DeepEqual(got, want) {
		t.Fatalf("RABBIT = %v", got)
	}
}

// Mientras el gateway siga publicando, un servicio sin la variable funciona igual y el
// valor del KV llega sin tocar (mismo texto, no un JSON re-serializado).
func TestConSecretosSinVariableUsaElKV(t *testing.T) {
	raw := `{"DB_PASSWORD":"kv","DB_HOST":"db"}`
	completo, faltan := ConSecretos("DB", raw, entorno(nil))
	if completo != raw || len(faltan) != 0 {
		t.Fatalf("DB = %q faltan=%v", completo, faltan)
	}
}

func TestConSecretosDBLocalUsaSuVariable(t *testing.T) {
	completo, _ := ConSecretos("DB_LOCAL", `{"DB_HOST":"db"}`, entorno(map[string]string{
		"DB_PASSWORD":       "prod",
		"DB_PASSWORD_LOCAL": "local",
	}))
	if got := campos(t, completo)["DB_PASSWORD"]; got != "local" {
		t.Fatalf("DB_LOCAL.DB_PASSWORD = %q", got)
	}
}

// El caso que rompe: secreto quitado del KV y sin variable. Se avisa cual falta.
func TestConSecretosAvisaLoQueFalta(t *testing.T) {
	_, faltan := ConSecretos("GMAILSENDER", `{"MAIL":"a@b"}`, entorno(nil))
	if !reflect.DeepEqual(faltan, []string{"GMAILSENDER_PASS"}) {
		t.Fatalf("faltan = %v", faltan)
	}
	if _, faltan := ConSecretos("JWT", "", entorno(nil)); !reflect.DeepEqual(faltan, []string{"JWT_KEY"}) {
		t.Fatalf("JWT faltan = %v", faltan)
	}
}

func TestConSecretosJWTDelEntorno(t *testing.T) {
	if got, _ := ConSecretos("JWT", "", entorno(map[string]string{"JWT_KEY": "k"})); got != "k" {
		t.Fatalf("JWT = %q", got)
	}
	if got, _ := ConSecretos("JWT", "kv", entorno(nil)); got != "kv" {
		t.Fatalf("JWT sin variable = %q", got)
	}
}

func TestConSecretosJSONInvalidoSeDevuelveIgual(t *testing.T) {
	if got, _ := ConSecretos("DB", "no-json", entorno(map[string]string{"DB_PASSWORD": "x"})); got != "no-json" {
		t.Fatalf("DB = %q", got)
	}
}

func TestRegistrationWorkerConTTL(t *testing.T) {
	reg, err := Ficha{
		FichaID:        "medfri-traza",
		FichaName:      "medfri-traza",
		FichaPort:      "9010",
		FichaHealthTTL: "30s",
	}.Registration()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Check == nil || reg.Check.TTL != (30*time.Second).String() || reg.Check.HTTP != "" {
		t.Fatalf("check = %+v", reg.Check)
	}
	if reg.Check.CheckID != "service:medfri-traza:ttl" {
		t.Fatalf("CheckID = %q", reg.Check.CheckID)
	}
}

func TestRegistrationHealthPathGanaAlTTL(t *testing.T) {
	reg, err := Ficha{
		FichaName:       "medfri-pai360",
		FichaAddress:    "172.17.0.1",
		FichaPort:       "9060",
		FichaHealthPath: "/medfri-pai360/health",
		FichaHealthTTL:  "30s",
	}.Registration()
	if err != nil {
		t.Fatal(err)
	}
	if reg.Check.HTTP == "" || reg.Check.TTL != "" {
		t.Fatalf("check = %+v", reg.Check)
	}
}

func TestRegistrationTTLInvalido(t *testing.T) {
	for _, ttl := range []string{"treinta", "1s"} {
		if _, err := (Ficha{FichaPort: "9010", FichaHealthTTL: ttl}).Registration(); err == nil {
			t.Errorf("TTL %q deberia ser error", ttl)
		}
	}
}
