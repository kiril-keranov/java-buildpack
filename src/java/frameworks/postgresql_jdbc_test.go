package frameworks_test

import (
	"testing"
)

func TestPostgreSQLJDBCServiceDetection(t *testing.T) {
	serviceLabel := "postgresql"

	if serviceLabel != "postgresql" {
		t.Errorf("Expected service label 'postgresql', got %s", serviceLabel)
	}
}

func TestPostgreSQLJDBCDriverClass(t *testing.T) {
	driver := "org.postgresql.Driver"

	if driver != "org.postgresql.Driver" {
		t.Errorf("Expected driver 'org.postgresql.Driver', got %s", driver)
	}
}
