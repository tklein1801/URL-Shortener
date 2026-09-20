package config

import "testing"

func TestParse(t *testing.T) {
	c, err := Parse(func(string) string { return "" })
	if err != nil || c.TokenFile != "./data/master-token" || c.SQLitePath != "./data/links.db" || c.Addr != ":3000" {
		t.Fatalf("%+v %v", c, err)
	}
	custom, err := Parse(func(key string) string {
		if key == "SQLITE_PATH" {
			return "custom/links.sqlite"
		}
		return ""
	})
	if err != nil || custom.SQLitePath != "custom/links.sqlite" {
		t.Fatalf("custom SQLite path: %+v %v", custom, err)
	}
	for key, value := range map[string]string{"PORT": "65536", "SQLITE_PATH": " ", "HTTP_READ_TIMEOUT": "0s", "HTTP_WRITE_TIMEOUT": "bad", "HTTP_IDLE_TIMEOUT": "-1s", "SHUTDOWN_TIMEOUT": "0", "BACKEND_TIMEOUT": "0s"} {
		t.Run(key, func(t *testing.T) {
			if _, err := Parse(func(k string) string {
				if k == key {
					return value
				}
				return ""
			}); err == nil {
				t.Fatal("accepted invalid configuration")
			}
		})
	}
}
