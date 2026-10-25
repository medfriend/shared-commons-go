package consul

import "testing"

// Ficha de un worker como las que publica getway-go hoy: sin SERVICE_ADDRESS. Debe
// registrarse igual que antes, sin chequeo.
func TestRegistrationFichaSinDireccionSigueRegistrando(t *testing.T) {
	reg, err := Ficha{
		FichaID:        "medfri-traza",
		FichaName:      "medfri-traza",
		FichaPort:      "9010",
		"SERVICE_PATH": "",
	}.Registration()
	if err != nil {
		t.Fatalf("ficha sin direccion: %v", err)
	}
	if reg.Address != "" || reg.Port != 9010 || reg.Check != nil {
		t.Fatalf("registro inesperado: %+v", reg)
	}
}

func TestRegistrationConHealthPathAgregaChequeo(t *testing.T) {
	reg, err := Ficha{
		FichaID:         "medfri-olap",
		FichaName:       "medfri-olap",
		FichaAddress:    "172.17.0.1",
		FichaPort:       "9080",
		FichaHealthPath: "medfri-olap/health",
	}.Registration()
	if err != nil {
		t.Fatalf("registration: %v", err)
	}
	if reg.Check == nil {
		t.Fatal("falta el chequeo HTTP")
	}
	if reg.Check.HTTP != "http://172.17.0.1:9080/medfri-olap/health" {
		t.Fatalf("url del chequeo = %q", reg.Check.HTTP)
	}
	if reg.Check.CheckID != "service:medfri-olap:http" || reg.Check.DeregisterCriticalServiceAfter == "" {
		t.Fatalf("chequeo incompleto: %+v", reg.Check)
	}
}

func TestRegistrationHealthPathSinDireccionEsError(t *testing.T) {
	_, err := Ficha{
		FichaName:       "medfri-olap",
		FichaPort:       "9080",
		FichaHealthPath: "/health",
	}.Registration()
	if err == nil {
		t.Fatal("un chequeo sin direccion no puede apuntar a ningun lado")
	}
}

func TestRegistrationPuertoInvalido(t *testing.T) {
	for _, port := range []string{"", "abc", "0", "-1"} {
		if _, err := (Ficha{FichaName: "x", FichaPort: port}).Registration(); err == nil {
			t.Fatalf("puerto %q deberia fallar", port)
		}
	}
}

func TestServiceIDUsaNombreSiNoHayID(t *testing.T) {
	reg, err := Ficha{FichaName: "medfri-cron", FichaPort: "9070"}.Registration()
	if err != nil {
		t.Fatal(err)
	}
	if got := serviceID(reg); got != "medfri-cron" {
		t.Fatalf("serviceID = %q", got)
	}
}

func TestParseFicha(t *testing.T) {
	if _, err := ParseFicha("PAI360", ""); err == nil {
		t.Fatal("clave vacia deberia fallar")
	}
	if _, err := ParseFicha("PAI360", `{"SERVICE_PORT": 9060}`); err == nil {
		t.Fatal("un numero no es texto: deberia fallar")
	}
	f, err := ParseFicha("PAI360", `{"SERVICE_PORT":"9060"}`)
	if err != nil || f[FichaPort] != "9060" {
		t.Fatalf("ficha = %v, err = %v", f, err)
	}
}

func TestKeyForService(t *testing.T) {
	casos := map[string]string{
		"medfri-pai360":       "PAI360",
		"medfri-olap":         "OLAP",
		"medfri-file-convert": "FILE", // segundo segmento, igual que la version anterior
		"security":            "SECURITY",
		"medfri-":             "MEDFRI-",
	}
	for in, want := range casos {
		if got := KeyForService(in); got != want {
			t.Errorf("KeyForService(%q) = %q, quiero %q", in, got, want)
		}
	}
}
