package config

import (
	"os"
	"strings"
	"testing"
)

// fixedByImage are settings the Dockerfile sets, so compose leaves them alone.
var fixedByImage = map[string]bool{"SCABBAGE_ADDR": true, "SCABBAGE_DATA_DIR": true, "SCABBAGE_WEB_DIR": true}

// envKeys lists every environment variable Load reads.
func envKeys(t *testing.T) []string {
	var keys []string
	if _, err := Load(nil, func(k string) string { keys = append(keys, k); return "" }); err != nil {
		t.Fatal(err)
	}
	return keys
}

func TestDockerFilesOfferEverySetting(t *testing.T) {
	for _, file := range []string{"../../../compose.yaml", "../../../.env.example"} {
		text, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, k := range envKeys(t) {
			if !fixedByImage[k] && !strings.Contains(string(text), k+":") && !strings.Contains(string(text), k+"=") {
				t.Errorf("%s doesn't offer %s", file, k)
			}
		}
	}
}
