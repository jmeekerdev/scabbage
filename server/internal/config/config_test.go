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
	want := Config{Addr: ":8080", WebDir: "web", DataDir: "data", ServerName: "Scabbage", TickRate: 20,
		Backups: 7, BackupHour: 4, Mode: "creative", SpawnZone: 16}
	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got %+v, want %+v", c, want)
	}
}

func TestEnvOverridesDefaults(t *testing.T) {
	c, err := Load(nil, env(map[string]string{
		"SCABBAGE_ADDR":     ":9000",
		"SCABBAGE_DATA_DIR": "/data",
		"SCABBAGE_ORIGINS":  "localhost:5173, 192.168.1.5:5173",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":9000" || c.DataDir != "/data" {
		t.Fatalf("addr = %q, data dir = %q", c.Addr, c.DataDir)
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

func TestBackupSettings(t *testing.T) {
	c, err := Load([]string{"-backups", "0"}, env(map[string]string{"SCABBAGE_BACKUP_HOUR": "23"}))
	if err != nil {
		t.Fatal(err)
	}
	if c.Backups != 0 || c.BackupHour != 23 {
		t.Fatalf("backups = %d, hour = %d", c.Backups, c.BackupHour)
	}
}

func TestInvalidBackupHour(t *testing.T) {
	if _, err := Load([]string{"-backup-hour", "24"}, env(nil)); err == nil {
		t.Fatal("expected error for hour 24")
	}
}

func TestSpawnZone(t *testing.T) {
	c, err := Load([]string{"-spawn-zone", "0"}, env(nil))
	if err != nil || c.SpawnZone != 0 {
		t.Fatalf("spawn zone = %d, err = %v", c.SpawnZone, err)
	}
	if _, err := Load(nil, env(map[string]string{"SCABBAGE_SPAWN_ZONE": "-1"})); err == nil {
		t.Fatal("expected error for a negative spawn zone")
	}
}

func TestGameMode(t *testing.T) {
	c, err := Load([]string{"-mode", "survival"}, env(nil))
	if err != nil || c.Mode != "survival" {
		t.Fatalf("mode = %q, err = %v", c.Mode, err)
	}
	if _, err := Load(nil, env(map[string]string{"SCABBAGE_MODE": "flying"})); err == nil {
		t.Fatal("expected error for an unknown mode")
	}
}

func TestUpgradeWorldFromFlagOrEnv(t *testing.T) {
	byFlag, err := Load([]string{"-upgrade-world"}, env(nil))
	if err != nil || !byFlag.Upgrade {
		t.Fatalf("flag: %+v, %v", byFlag, err)
	}
	byEnv, err := Load(nil, env(map[string]string{"SCABBAGE_UPGRADE_WORLD": "1"}))
	if err != nil || !byEnv.Upgrade {
		t.Fatalf("env: %+v, %v", byEnv, err)
	}
}
