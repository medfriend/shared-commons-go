package consul

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

// Claves de la ficha que getway-go publica en el KV por servicio (PAI360, CRON, OLAP...).
// SERVICE_PATH existe por compatibilidad y hoy repite SERVICE_ADDRESS.
const (
	FichaID         = "SERVICE_ID"
	FichaName       = "SERVICE_NAME"
	FichaAddress    = "SERVICE_ADDRESS"
	FichaPort       = "SERVICE_PORT"
	FichaHealthPath = "SERVICE_HEALTH_PATH"
	// FichaHealthTTL (ej. "30s") es el chequeo de los workers sin HTTP: el proceso avisa
	// a Consul cada tercio del TTL; si muere, el chequeo vence y pasa a critical.
	FichaHealthTTL = "SERVICE_HEALTH_TTL"
)

// Valores del chequeo de salud. DeregisterAfter saca del catalogo una instancia que
// lleva ese tiempo caida; al volver a arrancar se registra de nuevo.
const (
	healthInterval        = "10s"
	healthTimeout         = "3s"
	healthDeregisterAfter = "2m"
)

// Ficha es el JSON que guarda una clave de servicio en el KV.
type Ficha map[string]string

// ParseFicha decodifica una ficha del KV. Un JSON vacio o invalido es error: antes se
// ignoraba y el servicio arrancaba con un mapa nil.
func ParseFicha(key string, raw string) (Ficha, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("consul: la clave %s esta vacia", key)
	}
	var ficha Ficha
	if err := json.Unmarshal([]byte(raw), &ficha); err != nil {
		return nil, fmt.Errorf("consul: la clave %s no es un JSON de texto a texto: %w", key, err)
	}
	return ficha, nil
}

// Registration arma el registro del agente a partir de la ficha. Solo SERVICE_PORT es
// obligatorio, como antes: varias fichas de workers (TRAZA, WHATSAPP, FILEMAKER...) no
// traen SERVICE_ADDRESS y se registran sin direccion; exigirla los tumbaria al arrancar.
//
// El chequeo HTTP solo se agrega si la ficha trae SERVICE_HEALTH_PATH: un servicio sin
// ese campo queda como hasta ahora (sin chequeo, siempre "passing"), asi nada deja de
// enrutarse por cambiar la libreria. Sin health path, SERVICE_HEALTH_TTL agrega un
// chequeo por latido que no necesita direccion (workers de RabbitMQ/ZMQ).
func (f Ficha) Registration() (*api.AgentServiceRegistration, error) {
	port, err := strconv.Atoi(f[FichaPort])
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("consul: SERVICE_PORT invalido %q", f[FichaPort])
	}

	reg := &api.AgentServiceRegistration{
		ID:      f[FichaID],
		Name:    f[FichaName],
		Address: f[FichaAddress],
		Port:    port,
	}

	if path := strings.TrimSpace(f[FichaHealthPath]); path != "" {
		if reg.Address == "" || reg.Name == "" {
			return nil, fmt.Errorf("consul: SERVICE_HEALTH_PATH necesita SERVICE_NAME y SERVICE_ADDRESS")
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		reg.Check = &api.AgentServiceCheck{
			CheckID:                        "service:" + serviceID(reg) + ":http",
			Name:                           reg.Name + " health",
			HTTP:                           fmt.Sprintf("http://%s:%d%s", reg.Address, port, path),
			Method:                         "GET",
			Interval:                       healthInterval,
			Timeout:                        healthTimeout,
			DeregisterCriticalServiceAfter: healthDeregisterAfter,
		}
	} else if ttl, ok, err := f.healthTTL(); err != nil {
		return nil, err
	} else if ok {
		reg.Check = &api.AgentServiceCheck{
			CheckID:                        ttlCheckID(serviceID(reg)),
			Name:                           serviceID(reg) + " latido",
			TTL:                            ttl.String(),
			DeregisterCriticalServiceAfter: healthDeregisterAfter,
		}
	}
	return reg, nil
}

// healthTTL lee SERVICE_HEALTH_TTL. ok=false si no esta; menos de 3 s es error porque
// el latido va cada tercio y Consul no lo distingue de la red.
func (f Ficha) healthTTL() (time.Duration, bool, error) {
	raw := strings.TrimSpace(f[FichaHealthTTL])
	if raw == "" {
		return 0, false, nil
	}
	ttl, err := time.ParseDuration(raw)
	if err != nil || ttl < 3*time.Second {
		return 0, false, fmt.Errorf("consul: SERVICE_HEALTH_TTL invalido %q (minimo 3s)", raw)
	}
	return ttl, true, nil
}

func ttlCheckID(id string) string {
	return "service:" + id + ":ttl"
}

// Latir marca el chequeo TTL como passing cada ttl/3 hasta que el proceso termina. El
// primer latido va enseguida: el chequeo nace critical hasta recibirlo.
func Latir(client *api.Client, id string, ttl time.Duration, onError func(error)) {
	checkID := ttlCheckID(id)
	tick := time.NewTicker(ttl / 3)
	defer tick.Stop()
	for {
		if err := client.Agent().UpdateTTL(checkID, "ok", api.HealthPassing); err != nil && onError != nil {
			onError(err)
		}
		<-tick.C
	}
}

// serviceID es el ID con que queda el registro: Consul usa el nombre si no hay ID.
func serviceID(reg *api.AgentServiceRegistration) string {
	if reg.ID != "" {
		return reg.ID
	}
	return reg.Name
}

// Register registra el servicio de la ficha en el agente y devuelve su ID, que es lo que
// necesita Deregister al apagar.
func Register(client *api.Client, ficha Ficha) (string, error) {
	reg, err := ficha.Registration()
	if err != nil {
		return "", err
	}
	if err := client.Agent().ServiceRegister(reg); err != nil {
		return "", fmt.Errorf("consul: registrando %s: %w", serviceID(reg), err)
	}
	return serviceID(reg), nil
}

// Deregister saca la instancia del catalogo. Se llama al recibir SIGTERM para que el
// gateway deje de enrutar antes de que el proceso muera.
func Deregister(client *api.Client, id string) error {
	if id == "" {
		return fmt.Errorf("consul: deregister sin service id")
	}
	if err := client.Agent().ServiceDeregister(id); err != nil {
		return fmt.Errorf("consul: deregister %s: %w", id, err)
	}
	return nil
}

// GetFicha lee y decodifica una clave de servicio del KV.
func GetFicha(client *api.Client, key string) (Ficha, error) {
	raw, err := GetKeyValue(client, key)
	if err != nil {
		return nil, err
	}
	return ParseFicha(key, raw)
}

// KeyForService traduce el nombre de catalogo a su clave de KV: medfri-pai360 -> PAI360.
// Toma el segundo segmento, como siempre lo hizo GetServiceAddressAndPort; sin guion
// devuelve el nombre entero en mayusculas (antes eso era un panic).
func KeyForService(serviceName string) string {
	parts := strings.Split(serviceName, "-")
	if len(parts) > 1 && parts[1] != "" {
		return strings.ToUpper(parts[1])
	}
	return strings.ToUpper(serviceName)
}
