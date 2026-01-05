package frameworks_test

import (
	"encoding/xml"
	"testing"
)

type ContrastConfig struct {
	XMLName     xml.Name `xml:"contrast"`
	ID          string   `xml:"id"`
	GlobalKey   string   `xml:"global-key"`
	URL         string   `xml:"url"`
	ResultsMode string   `xml:"results-mode"`
}

func TestContrastSecurityConfigXMLStructure(t *testing.T) {
	xmlConfig := `<?xml version="1.0" encoding="UTF-8"?>
<contrast>
  <id>default</id>
  <global-key>test-api-key</global-key>
  <url>https://app.contrastsecurity.com/Contrast/s/</url>
  <results-mode>never</results-mode>
</contrast>`

	var config ContrastConfig
	if err := xml.Unmarshal([]byte(xmlConfig), &config); err != nil {
		t.Fatalf("Failed to parse Contrast Security XML config: %v", err)
	}

	if config.ID != "default" {
		t.Errorf("Expected ID 'default', got %s", config.ID)
	}
	if config.ResultsMode != "never" {
		t.Errorf("Expected results-mode 'never', got %s", config.ResultsMode)
	}
}

func TestContrastSecurityCredentialKeys(t *testing.T) {
	credentials := map[string]interface{}{
		"api_key":        "test-api-key-123",
		"service_key":    "test-service-key-456",
		"teamserver_url": "https://app.contrastsecurity.com",
		"username":       "test@example.com",
	}

	requiredKeys := []string{"api_key", "service_key", "teamserver_url", "username"}
	for _, key := range requiredKeys {
		if _, exists := credentials[key]; !exists {
			t.Errorf("Required credential key %s is missing", key)
		}
	}
}
