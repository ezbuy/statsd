package statsd

import (
	"fmt"
	"log"
	"time"
)

// Config is the struct to run statsd
type Config struct {
	Host       string
	Port       int
	Project    string
	Enable     bool    // flag used to indicate whether stats is enabled
	SampleRate float32 // global statsd sample rate
}

const (
	defaultSampleRate = 1.0
)

var config *Config
var addr string

// Setup set the config
func Setup(cfg *Config) {
	config = cfg

	// if sample rate is equal to 0, it indicates that the statsd never called
	// so we set a default value
	if config.SampleRate == 0 {
		config.SampleRate = defaultSampleRate
	}

	addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
}

type metricType int16

const (
	metricTypeCount metricType = iota
	metricTypeGauge
	metricTypeFGauge
	metricTypeTimer
)

// Incr increment a particular event
func Incr(stat string) {
	IncrByVal(stat, 1)
}

// IncrByVal increment a particular event with value
func IncrByVal(stat string, val int) {
	// check whether is initialized
	if config == nil {
		return
	}

	IncrWithSampling(stat, val, config.SampleRate)
}

// IncrWithSampling increment a particular event with value and sampling
func IncrWithSampling(stat string, val int, sampleRate float32) {
	if config == nil {
		return
	}

	if !config.Enable {
		return
	}

	if val == 0 {
		return // ignore
	}

	send(stat, val, metricTypeCount, sampleRate)
}

// Gauge set a constant value of a particular event
func Gauge(stat string, val int) {
	if config == nil {
		return
	}

	GaugeWithSampling(stat, val, config.SampleRate)
}

// GaugeWithSampling set a constant value of a particular event with sampling
func GaugeWithSampling(stat string, val int, sampleRate float32) {
	if config == nil {
		return
	}

	if !config.Enable {
		return
	}

	gauge(stat, val, metricTypeGauge, sampleRate)
}

// FGauge set a constant float point value of a particular event
func FGauge(stat string, val float64) {
	if config == nil {
		return
	}

	FGaugeWithSampling(stat, val, config.SampleRate)
}

// FGaugeWithSampling set a constant float point value of a particular event with sampling
func FGaugeWithSampling(stat string, val float64, sampleRate float32) {
	if config == nil {
		return
	}

	if !config.Enable {
		return
	}

	gauge(stat, val, metricTypeFGauge, sampleRate)
}

func gauge(stat string, val interface{}, t metricType, sampleRate float32) {
	send(stat, val, t, sampleRate)
}

// TimingByValue track duration of a event
func TimingByValue(stat string, d time.Duration) {
	if config == nil {
		return
	}

	TimingByValueWithSampling(stat, d, config.SampleRate)
}

// TimingByValueWithSampling track duration of a event with sampling
func TimingByValueWithSampling(stat string, d time.Duration, sampleRate float32) {
	if config == nil {
		return
	}

	if !config.Enable {
		return
	}

	// the delta must be given in milliseconds
	t := d / time.Millisecond

	send(stat, int64(t), metricTypeTimer, sampleRate)
}

// Timing track duration of a event
func Timing(stat string, t1 time.Time, t2 time.Time) {
	if config == nil {
		return
	}

	TimingWithSampling(stat, t1, t2, config.SampleRate)
}

// TimingWithSampling track duration of a event with sampling
func TimingWithSampling(stat string, t1 time.Time, t2 time.Time, sampleRate float32) {
	TimingByValueWithSampling(stat, t2.Sub(t1), sampleRate)
}

// Now return current system time
func Now() time.Time {
	return time.Now()
}

func send(stat string, val interface{}, t metricType, sampleRate float32) {
	if stat == "" {
		return
	}

	// error handling
	defer func() {
		if err := recover(); err != nil {
			log.Printf("[statsd] %s", err)
		}
	}()

	client, err := newClient(addr, config.Project)
	if err != nil {
		panic(err)
	}

	defer client.Close()

	switch t {
	case metricTypeCount:
		if i, ok := val.(int); ok {
			client.IncrWithSampling(stat, i, sampleRate)
		}
	case metricTypeGauge:
		if i, ok := val.(int); ok {
			client.GaugeWithSampling(stat, i, sampleRate)
		}
	case metricTypeFGauge:
		if i, ok := val.(float64); ok {
			client.FGaugeWithSampling(stat, i, sampleRate)
		}
	case metricTypeTimer:
		if i, ok := val.(int64); ok {
			client.TimingWithSampling(stat, i, sampleRate)
		}
	default:
		// temporary do nothing
	}
}
