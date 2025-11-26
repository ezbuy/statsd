package statsd

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
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
	addr   string
	prefix string
	conn   net.Conn

	buffer      []countBuffer
	m           sync.Mutex
	flushticker *time.Ticker
}

type countBuffer struct {
	name  string
	count int64
}

func newClient(addr string, prefix string) (*Client, error) {
	prefix = strings.TrimRight(prefix, ".")

	c := &Client{
		addr:   addr,
		prefix: prefix,
	}

	conn, err := net.DialTimeout("udp", addr, 5*time.Second)
	if err != nil {
		return nil, err
	}

	c.conn = conn
	c.flushticker = time.NewTicker(5 * time.Second)
	go c.bufferSendLoop()

	return c, nil
}

func (c *Client) bufferSendLoop() {
	for range c.flushticker.C {
		c.m.Lock()
		if len(c.buffer) == 0 {
			c.m.Unlock()
			continue
		}
		buffer := c.buffer
		c.buffer = nil
		c.m.Unlock()
		for idx := range buffer {
			c.send(buffer[idx].name, buffer[idx].count, "c", 1)
		}
	}
}

func (c *Client) addToBuffer(stat string, count int64) {
	c.m.Lock()
	for i := range c.buffer {
		if c.buffer[i].name == stat {
			c.buffer[i].count++
			c.m.Unlock()
			return
		}
	}
	c.buffer = append(c.buffer, countBuffer{stat, count})
	c.m.Unlock()
}

// Close the UDP connection
func (c *Client) Close() error {
	c.flushticker.Stop()
	if c.conn == nil {
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

	c.addToBuffer(stat, count)
	// return c.send(stat, count, "c", sampleRate)
	return nil
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

	return c.send(stat, -count, "c", sampleRate)
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

	return c.send(stat, delta, "ms", sampleRate)
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
		c.send(stat, 0, "g", 1)
	}

	return c.send(stat, value, "g", sampleRate)
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
		c.send(stat, 0, "g", 1)
	}

	return c.send(stat, value, "g", sampleRate)
}

// write a UDP packet with the statsd event
func (c *Client) send(bucket string, value interface{}, t string, sampleRate float32) error {
	if c.conn == nil {
		return ErrNotConnected
	}

	if c.prefix != "" {
		bucket = fmt.Sprintf("%s.%s", c.prefix, bucket)
	}

	metric := fmt.Sprintf("%s:%v|%s|@%f", bucket, value, t, sampleRate)

	_, err := c.conn.Write([]byte(metric))
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

var fireRand *rand.Rand

func init() {
	// use rand.Seed to reset fireRand regularly?
	fireRand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func shouldFire(sampleRate float32) bool {
	if sampleRate == 1 {
		return true
	}

	// r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// return r.Float32() <= sampleRate
	return fireRand.Float32() <= sampleRate
}
