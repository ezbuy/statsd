package statsd

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"
)

// errors
var (
	ErrNotConnected      = errors.New("cannot send stats, not connected to StatsD server")
	ErrInvalidCount      = errors.New("count is less than 0")
	ErrInvalidSampleRate = errors.New("sample rate is larger than 1 or less then 0")
)

// Client is a client library to send events to StatsD
type Client struct {
	conn           net.Conn
	addr           string
	prefix         string
	eventStringTpl string
}

func newClient(addr string, prefix string) *Client {
	return &Client{
		addr:           addr,
		prefix:         prefix,
		eventStringTpl: "%s%s:%s",
	}
}

// String returns the StatsD server address
func (c *Client) String() string {
	return c.addr
}

// CreateSocket creates a UDP connection to a StatsD server
func (c *Client) CreateSocket() error {
	conn, err := net.DialTimeout("udp", c.addr, 5*time.Second)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Close the UDP connection
func (c *Client) Close() error {
	if nil == c.conn {
		return nil
	}
	return c.conn.Close()
}

// See statsd data types here: http://statsd.readthedocs.org/en/latest/types.html
// or also https://github.com/b/statsd_spec

// Incr - Increment a counter metric. Often used to note a particular event
func (c *Client) Incr(stat string, count int64) error {
	return c.IncrWithSampling(stat, count, 1)
}

// IncrWithSampling - Increment a counter metric with sampling between 0 and 1
func (c *Client) IncrWithSampling(stat string, count int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	if err := checkCount(count); err != nil {
		return err
	}

	return c.send(stat, "%d|c", count, sampleRate)
}

// Decr - Decrement a counter metric. Often used to note a particular event
func (c *Client) Decr(stat string, count int64) error {
	return c.DecrWithSampling(stat, count, 1)
}

// DecrWithSampling - Decrement a counter metric with sampling between 0 and 1
func (c *Client) DecrWithSampling(stat string, count int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	if err := checkCount(count); err != nil {
		return err
	}

	return c.send(stat, "%d|c", -count, sampleRate)
}

// Timing - Track a duration event
// the time delta must be given in milliseconds
func (c *Client) Timing(stat string, delta int64) error {
	return c.TimingWithSampling(stat, delta, 1)
}

// TimingWithSampling track a duration event with sampling between 0 and 1
func (c *Client) TimingWithSampling(stat string, delta int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	return c.send(stat, "%d|ms", delta, sampleRate)
}

// Gauge - Gauges are a constant data type. They are not subject to averaging,
// and they don’t change unless you change them. That is, once you set a gauge value,
// it will be a flat line on the graph until you change it again. If you specify
// delta to be true, that specifies that the gauge should be updated, not set. Due to the
// underlying protocol, you can't explicitly set a gauge to a negative number without
// first setting it to zero.
func (c *Client) Gauge(stat string, value int64) error {
	return c.GaugeWithSampling(stat, value, 1)
}

// GaugeWithSampling set a constant data type with sampling between 0 and 1
func (c *Client) GaugeWithSampling(stat string, value int64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil // ignore this call
	}

	if value < 0 {
		c.send(stat, "%d|g", 0, 1)
	}

	return c.send(stat, "%d|g", value, sampleRate)
}

// FGauge -- Send a floating point value for a gauge
func (c *Client) FGauge(stat string, value float64) error {
	return c.FGaugeWithSampling(stat, value, 1)
}

// FGaugeWithSampling send a floating point value for a gauge with sampling between 0 and 1
func (c *Client) FGaugeWithSampling(stat string, value float64, sampleRate float32) error {
	if err := checkSampleRate(sampleRate); err != nil {
		return err
	}

	if !shouldFire(sampleRate) {
		return nil
	}

	if value < 0 {
		c.send(stat, "%d|g", 0, 1)
	}

	return c.send(stat, "%g|g", value, sampleRate)
}

// write a UDP packet with the statsd event
func (c *Client) send(stat string, format string, value interface{}, sampleRate float32) error {
	if c.conn == nil {
		return ErrNotConnected
	}

	format = fmt.Sprintf(c.eventStringTpl, c.prefix, stat, format)

	if sampleRate != 1 {
		format = fmt.Sprintf("%s|@%f", format, sampleRate)
	}

	_, err := fmt.Fprintf(c.conn, format, value)
	return err
}

func checkCount(c int64) error {
	if c <= 0 {
		return ErrInvalidCount
	}

	return nil
}

func checkSampleRate(r float32) error {
	if r < 0 || r > 1 {
		return ErrInvalidSampleRate
	}

	return nil
}

func shouldFire(sampleRate float32) bool {
	if sampleRate == 1 {
		return true
	}

	r := rand.New(rand.NewSource(time.Now().Unix()))

	return r.Float32() <= sampleRate
}
