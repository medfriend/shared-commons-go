package consul

import (
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

// Instance es una instancia sana de un servicio: a donde se le habla.
type Instance struct {
	Address string
	Port    int
}

// LookupFunc devuelve las instancias sanas de un servicio del catalogo.
type LookupFunc func(serviceName string) ([]Instance, error)

// HealthyInstances consulta el catalogo de salud con passingOnly: una instancia con su
// chequeo en critical no se devuelve. Las instancias sin chequeo cuentan como sanas,
// asi que los servicios que aun no declaran SERVICE_HEALTH_PATH se siguen enrutando.
//
// Direccion: la del registro del servicio; si viene vacia, la SERVICE_ADDRESS de su
// ficha en el KV (lo que hacia la version anterior); y si tampoco hay, la del nodo.
func HealthyInstances(client *api.Client) LookupFunc {
	return func(serviceName string) ([]Instance, error) {
		entries, _, err := client.Health().Service(serviceName, "", true, nil)
		if err != nil {
			return nil, fmt.Errorf("consul: consultando salud de %s: %w", serviceName, err)
		}
		var fichaAddress *string
		kvAddress := func() string {
			if fichaAddress == nil {
				addr := ""
				if ficha, err := GetFicha(client, KeyForService(serviceName)); err == nil {
					addr = ficha[FichaAddress]
				}
				fichaAddress = &addr
			}
			return *fichaAddress
		}
		instances := make([]Instance, 0, len(entries))
		for _, e := range entries {
			if e == nil || e.Service == nil {
				continue
			}
			addr := e.Service.Address
			if addr == "" {
				addr = kvAddress()
			}
			if addr == "" && e.Node != nil {
				addr = e.Node.Address
			}
			if addr == "" || e.Service.Port <= 0 {
				continue
			}
			instances = append(instances, Instance{Address: addr, Port: e.Service.Port})
		}
		return instances, nil
	}
}

// Resolver recuerda las instancias de cada servicio durante ttl y reparte las peticiones
// entre ellas por turnos. El gateway lo usa para no ir a Consul en cada peticion.
type Resolver struct {
	lookup LookupFunc
	ttl    time.Duration
	now    func() time.Time

	mu      sync.Mutex
	entries map[string]*resolverEntry
}

type resolverEntry struct {
	instances []Instance
	expires   time.Time
	next      int
}

// NewResolver crea un resolvedor con cache. ttl <= 0 desactiva la cache.
func NewResolver(lookup LookupFunc, ttl time.Duration) *Resolver {
	return &Resolver{lookup: lookup, ttl: ttl, now: time.Now, entries: map[string]*resolverEntry{}}
}

// NewHealthResolver es NewResolver sobre el catalogo de salud de Consul.
func NewHealthResolver(client *api.Client, ttl time.Duration) *Resolver {
	return NewResolver(HealthyInstances(client), ttl)
}

// Resolve devuelve la siguiente instancia sana del servicio. Un error o una lista vacia
// no se guardan: la proxima llamada vuelve a preguntar.
func (r *Resolver) Resolve(serviceName string) (string, int, error) {
	r.mu.Lock()
	entry, ok := r.entries[serviceName]
	if ok && r.ttl > 0 && r.now().Before(entry.expires) {
		inst := entry.instances[entry.next%len(entry.instances)]
		entry.next++
		r.mu.Unlock()
		return inst.Address, inst.Port, nil
	}
	r.mu.Unlock()

	instances, err := r.lookup(serviceName)
	if err != nil {
		return "", 0, err
	}
	if len(instances) == 0 {
		return "", 0, fmt.Errorf("no healthy instances found for service: %s", serviceName)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[serviceName] = &resolverEntry{instances: instances, expires: r.now().Add(r.ttl), next: 1}
	return instances[0].Address, instances[0].Port, nil
}

// Invalidate olvida el servicio: se llama cuando una peticion a la instancia falla por
// conexion, para no insistir con una direccion vieja hasta que venza el ttl.
func (r *Resolver) Invalidate(serviceName string) {
	r.mu.Lock()
	delete(r.entries, serviceName)
	r.mu.Unlock()
}
