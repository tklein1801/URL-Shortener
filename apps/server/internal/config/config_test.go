package config

import "testing"

func TestParse(t *testing.T) {
	c, err := Parse(func(string) string { return "" })
	if err != nil || c.TokenFile != "./data/master-token" || c.RedisDB != 0 || c.Addr != ":3000" {
		t.Fatalf("%+v %v", c, err)
	}
	for key, value := range map[string]string{"PORT": "65536", "REDIS_DB": "-1", "REDIS_HOST": "invalid", "HTTP_READ_TIMEOUT": "0s", "HTTP_WRITE_TIMEOUT": "bad", "HTTP_IDLE_TIMEOUT": "-1s", "SHUTDOWN_TIMEOUT": "0", "BACKEND_TIMEOUT": "0s"} {
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
