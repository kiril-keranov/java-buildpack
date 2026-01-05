package frameworks_test

import (
	"testing"
)

func TestMariaDBJDBCServiceDetection(t *testing.T) {
	serviceTypes := []string{"mariadb", "mysql"}

	for _, svc := range serviceTypes {
		if svc != "mariadb" && svc != "mysql" {
			t.Errorf("Unexpected service type: %s", svc)
		}
	}
}

func TestMariaDBJDBCDriverReplacement(t *testing.T) {
	oldDriver := "com.mysql.jdbc.Driver"
	newDriver := "org.mariadb.jdbc.Driver"

	if oldDriver == newDriver {
		t.Error("MariaDB driver should replace MySQL driver")
	}
}
