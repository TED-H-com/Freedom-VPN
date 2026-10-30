package dnsfeeds

import (
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

type feedAge struct {
	code string
	at   time.Time
}
type feedMetrics struct {
	mu    sync.Mutex
	ages  map[uuid.UUID]feedAge
	now   func() time.Time
	total *prometheus.CounterVec
	age   *prometheus.Desc
}

func newFeedMetrics(now func() time.Time) *feedMetrics {
	return &feedMetrics{ages: map[uuid.UUID]feedAge{}, now: now,
		total: prometheus.NewCounterVec(prometheus.CounterOpts{Name: "fvpn_dns_feed_refresh_total", Help: "Committed refresh outcomes by active feed code and result."}, []string{"feed_code", "result"}),
		age:   prometheus.NewDesc("fvpn_dns_feed_last_good_age_seconds", "Seconds since last successful verification; +Inf before bootstrap. Maximum for duplicate active codes.", []string{"feed_code"}, nil)}
}
func (m *feedMetrics) Describe(ch chan<- *prometheus.Desc) { m.total.Describe(ch); ch <- m.age }
func (m *feedMetrics) Collect(ch chan<- prometheus.Metric) {
	m.total.Collect(ch)
	m.mu.Lock()
	defer m.mu.Unlock()
	ages := map[string]float64{}
	for _, f := range m.ages {
		age := math.Inf(1)
		if !f.at.IsZero() {
			age = max(0, m.now().Sub(f.at).Seconds())
		}
		ages[f.code] = max(ages[f.code], age)
	}
	for code, age := range ages {
		ch <- prometheus.MustNewConstMetric(m.age, prometheus.GaugeValue, age, code)
	}
}
func (m *feedMetrics) set(id uuid.UUID, code string, at time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for oldID, age := range m.ages {
		if oldID != id && age.code == code {
			delete(m.ages, oldID)
		}
	}
	m.ages[id] = feedAge{code: code, at: at}
}
