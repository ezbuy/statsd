package statsd

import "testing"

func initConfig() {
	config := &Config{
		Host:   "192.168.199.61",
		Port:   8125,
		Enable: true,
	}

	Setup(config)
}

func TestGauge(t *testing.T) {
	initConfig()

	Gauge("myproj.login.count", 10)
}
