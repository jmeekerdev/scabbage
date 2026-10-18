package config

import (
	"reflect"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestDefaults(t *testing.T) {
	c, err := Load(nil, env(nil))
	if err != nil {
		t.Fatal(err)
	}
	want := Config{Addr: ":8080", WebDir: "web", ServerName: "Scabbage", TickRate: 20}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %+v, want %+v", c, want)
	}
}

func TestEnvOverridesDefaults(t *testing.T) {
	c, err := Load(nil, env(map[string]string{
		"SCABBAGE_ADDR":    ":9000",
		"SCABBAGE_ORIGINS": "localhost:5173, 192.168.1.5:5173",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":9000" {
		t.Fatalf("addr = %q", c.Addr)
	}
	if want := []string{"localhost:5173", "192.168.1.5:5173"}; !reflect.DeepEqual(c.Origins, want) {
		t.Fatalf("origins = %q, want %q", c.Origins, want)
	}
}

func TestFlagsOverrideEnv(t *testing.T) {
	c, err := Load([]string{"-addr", ":7000"}, env(map[string]string{"SCABBAGE_ADDR": ":9000"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":7000" {
		t.Fatalf("addr = %q, want :7000", c.Addr)
	}
}

func TestInvalidTickRate(t *testing.T) {
	if _, err := Load([]string{"-tick-rate", "0"}, env(nil)); err == nil {
		t.Fatal("expected error for tick rate 0")
	}
}
