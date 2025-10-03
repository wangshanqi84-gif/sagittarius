package local

import (
	"context"
	"runtime"
	"sync"
	"time"
)

//////////////////////////////////////
// 基础监控 GO runtime
//////////////////////////////////////

const (
	_sysMetricDefaultInterval = 10
)

type Metric struct {
	ctx         context.Context
	data        sysMetric
	intervalSec int64
	open        bool
	reportCh    chan string
}

type Option func(*Metric)

var _m *Metric
var _once sync.Once

func InitMetric(ctx context.Context, opts ...Option) *Metric {
	_once.Do(func() {
		_m = &Metric{
			ctx:         ctx,
			reportCh:    make(chan string),
			open:        true,
			intervalSec: _sysMetricDefaultInterval,
		}
		for _, opt := range opts {
			if opt != nil {
				opt(_m)
			}
		}
	})

	return _m
}

func (m *Metric) Start() {
	go func() {
		if !m.open {
			close(m.reportCh)
			return
		}
		for {
			var timer *time.Timer
			timer = time.NewTimer(time.Duration(m.intervalSec) * time.Second)
			select {
			case <-timer.C:
				timer.Stop()
				// 读取信息
				runtime.ReadMemStats(&m.data.current.MemStats)
				m.data.current.NumGoroutine = runtime.NumGoroutine()
				// 生成报告
				report := m.data.GetReport()
				// 发送报告
				m.reportCh <- report.Format()
				// 后处理
				m.data.before = m.data.current
			case <-m.ctx.Done():
				timer.Stop()
				return
			}
		}
	}()
}

func (m *Metric) Reports() chan string {
	return m.reportCh
}

func SetOpenFlag(isOpen bool) Option {
	return func(m *Metric) {
		m.open = isOpen
	}
}

func SetIntervalSec(interval int64) Option {
	return func(m *Metric) {
		m.intervalSec = interval
	}
}
