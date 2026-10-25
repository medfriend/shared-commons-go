package consul

import (
	"errors"
	"testing"
	"time"
)

type fakeLookup struct {
	calls     int
	instances []Instance
	err       error
}

func (f *fakeLookup) lookup(string) ([]Instance, error) {
	f.calls++
	return f.instances, f.err
}

func TestResolverCacheaDentroDelTTL(t *testing.T) {
	fake := &fakeLookup{instances: []Instance{{"10.0.0.1", 9060}}}
	r := NewResolver(fake.lookup, time.Minute)
	ahora := time.Unix(1000, 0)
	r.now = func() time.Time { return ahora }

	for i := 0; i < 3; i++ {
		addr, port, err := r.Resolve("medfri-pai360")
		if err != nil || addr != "10.0.0.1" || port != 9060 {
			t.Fatalf("resolve = %s:%d, %v", addr, port, err)
		}
	}
	if fake.calls != 1 {
		t.Fatalf("consultas a consul = %d, quiero 1", fake.calls)
	}

	ahora = ahora.Add(time.Minute)
	if _, _, err := r.Resolve("medfri-pai360"); err != nil {
		t.Fatal(err)
	}
	if fake.calls != 2 {
		t.Fatalf("vencido el ttl debe volver a consultar; consultas = %d", fake.calls)
	}
}

func TestResolverRepartePorTurnos(t *testing.T) {
	fake := &fakeLookup{instances: []Instance{{"a", 1}, {"b", 2}}}
	r := NewResolver(fake.lookup, time.Minute)

	var got []string
	for i := 0; i < 4; i++ {
		addr, _, _ := r.Resolve("svc")
		got = append(got, addr)
	}
	want := []string{"a", "b", "a", "b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("turnos = %v, quiero %v", got, want)
		}
	}
}

func TestResolverNoGuardaErroresNiVacios(t *testing.T) {
	fake := &fakeLookup{err: errors.New("consul caido")}
	r := NewResolver(fake.lookup, time.Minute)

	if _, _, err := r.Resolve("svc"); err == nil {
		t.Fatal("deberia propagar el error")
	}
	fake.err = nil
	if _, _, err := r.Resolve("svc"); err == nil {
		t.Fatal("sin instancias sanas deberia ser error")
	}
	fake.instances = []Instance{{"a", 1}}
	if addr, _, err := r.Resolve("svc"); err != nil || addr != "a" {
		t.Fatalf("tras recuperarse: %s, %v", addr, err)
	}
	if fake.calls != 3 {
		t.Fatalf("consultas = %d, quiero 3", fake.calls)
	}
}

func TestResolverInvalidateFuerzaConsulta(t *testing.T) {
	fake := &fakeLookup{instances: []Instance{{"a", 1}}}
	r := NewResolver(fake.lookup, time.Minute)

	r.Resolve("svc")
	r.Invalidate("svc")
	r.Resolve("svc")
	if fake.calls != 2 {
		t.Fatalf("consultas = %d, quiero 2", fake.calls)
	}
}

func TestResolverSinTTLConsultaSiempre(t *testing.T) {
	fake := &fakeLookup{instances: []Instance{{"a", 1}}}
	r := NewResolver(fake.lookup, 0)

	r.Resolve("svc")
	r.Resolve("svc")
	if fake.calls != 2 {
		t.Fatalf("consultas = %d, quiero 2", fake.calls)
	}
}
