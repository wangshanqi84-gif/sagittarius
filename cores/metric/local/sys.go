package local

import (
	"bytes"
	"fmt"
	"runtime"
)

type sysCycleData struct {
	// 周期内堆分配总量 只计算增加
	AllocTotal string
	// 框架堆空间分配
	Alloc string
	// 服务现在系统使用的内存
	Sys string
	// 被runtime监视的指针数
	Lookups string
	// 周期内golang分配堆变化
	HeapAlloc string
	// 周期内系统分配堆变化
	HeapSys string
	// 周期栈用量变化
	StackInuse string
	// 周期内系统分配栈变化
	StackSys string
	// 周期内GC导致暂停时常(纳秒)
	PauseTotalNs uint64
	// 周期内malloc次数
	Mallocs uint64
	// 服务回收的次数
	Frees uint64
	// 周期内完成GC次数
	NumGC uint32
	// 周期内goroutine数量变化
	NumGoroutine string
}

type sysData struct {
	// 堆分配总量 只计算增加
	AllocTotal string
	// 框架堆空间分配
	Alloc string
	// 服务现在系统使用的内存
	Sys string
	// 被runtime监视的指针数
	Lookups string
	// golang分配堆数量
	HeapAlloc string
	// 系统分配堆数量
	HeapSys string
	// malloc次数
	Mallocs uint64
	// 服务回收的次数
	Frees uint64
	// 栈用量
	StackInuse string
	// 系统分配栈数量
	StackSys string
	// GC导致暂停时常(纳秒)
	PauseTotalNs uint64
	// 完成GC次数
	NumGC uint32
	// GC消耗CPU情况
	GCCPUFraction string
	// goroutine数量
	NumGoroutine string
}

type reportData struct {
	sysCycleData
	sysData
}

func (r *reportData) Format() string {
	var buf bytes.Buffer
	buf.WriteString("|||||||||||||||||||||||||||||||||||||||||||||||||\n")
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString("|               |    Current    |     Cycle     |\n")
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|   AllocTotal  |%15s|%15s|\n", r.sysData.AllocTotal, r.sysCycleData.AllocTotal))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|     Alloc     |%15s|%15s|\n", r.sysData.Alloc, r.sysCycleData.Alloc))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|   HeapAlloc   |%15s|%15s|\n", r.sysData.HeapAlloc, r.sysCycleData.HeapAlloc))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|      Sys      |%15s|%15s|\n", r.sysData.Sys, r.sysCycleData.Sys))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|    HeapSys    |%15s|%15s|\n", r.sysData.HeapSys, r.sysCycleData.HeapSys))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|  StackInuse   |%15s|%15s|\n", r.sysData.StackInuse, r.sysCycleData.StackInuse))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|   StackSys    |%15s|%15s|\n", r.sysData.StackSys, r.sysCycleData.StackSys))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|    Lookups    |%15s|%15s|\n", r.sysData.Lookups, r.sysCycleData.Lookups))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|    Mallocs    |%15d|%15d|\n", r.sysData.Mallocs, r.sysCycleData.Mallocs))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|     Frees     |%15d|%15d|\n", r.sysData.Frees, r.sysCycleData.Frees))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("| PauseTotalNs  |%15d|%15d|\n", r.sysData.PauseTotalNs, r.sysCycleData.PauseTotalNs))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("|     NumGC     |%15d|%15d|\n", r.sysData.NumGC, r.sysCycleData.NumGC))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("| NumGoroutine  |%15s|%15s|\n", r.sysData.NumGoroutine, r.sysCycleData.NumGoroutine))
	buf.WriteString("|---------------|---------------|---------------|\n")
	buf.WriteString(fmt.Sprintf("| GCCPUFraction |%15s|               |\n", r.sysData.GCCPUFraction))
	buf.WriteString("|---------------|---------------|---------------|\n\n")
	return buf.String()
}

type sysMetricData struct {
	runtime.MemStats
	NumGoroutine int
}

type sysMetric struct {
	before  sysMetricData
	current sysMetricData
}

func (sm *sysMetric) GetReport() reportData {
	return reportData{
		sysCycleData: sm.getSysCycle(),
		sysData:      sm.getMem(),
	}
}

func (sm *sysMetric) getSysCycle() sysCycleData {
	d := sysCycleData{
		AllocTotal:   size2Human(sm.current.TotalAlloc - sm.before.TotalAlloc),
		PauseTotalNs: sm.current.PauseTotalNs - sm.before.PauseTotalNs,
		Mallocs:      sm.current.Mallocs - sm.before.Mallocs,
		Frees:        sm.current.Frees - sm.before.Frees,
		NumGC:        sm.current.NumGC - sm.before.NumGC,
	}
	if sm.current.Alloc >= sm.before.Alloc {
		d.Alloc = size2Human(sm.current.Alloc - sm.before.Alloc)
	} else {
		d.Alloc = fmt.Sprintf("-%s", size2Human(sm.before.Alloc-sm.current.Alloc))
	}
	if sm.current.Sys >= sm.before.Sys {
		d.Sys = size2Human(sm.current.Sys - sm.before.Sys)
	} else {
		d.Sys = fmt.Sprintf("-%s", size2Human(sm.before.Sys-sm.current.Sys))
	}
	if sm.current.Lookups >= sm.before.Lookups {
		d.Lookups = size2Human(sm.current.Lookups - sm.before.Lookups)
	} else {
		d.Lookups = fmt.Sprintf("-%s", size2Human(sm.before.Lookups-sm.current.Lookups))
	}
	if sm.current.HeapAlloc >= sm.before.HeapAlloc {
		d.HeapAlloc = size2Human(sm.current.HeapAlloc - sm.before.HeapAlloc)
	} else {
		d.HeapAlloc = fmt.Sprintf("-%s", size2Human(sm.before.HeapAlloc-sm.current.HeapAlloc))
	}
	if sm.current.HeapSys >= sm.before.HeapSys {
		d.HeapSys = size2Human(sm.current.HeapSys - sm.before.HeapSys)
	} else {
		d.HeapSys = fmt.Sprintf("-%s", size2Human(sm.before.HeapSys-sm.current.HeapSys))
	}
	if sm.current.StackInuse >= sm.before.StackInuse {
		d.StackInuse = size2Human(sm.current.StackInuse - sm.before.StackInuse)
	} else {
		d.StackInuse = fmt.Sprintf("-%s", size2Human(sm.before.StackInuse-sm.current.StackInuse))
	}
	if sm.current.StackSys >= sm.before.StackSys {
		d.StackSys = size2Human(sm.current.StackSys - sm.before.StackSys)
	} else {
		d.StackSys = fmt.Sprintf("-%s", size2Human(sm.before.StackSys-sm.current.StackSys))
	}
	if sm.current.NumGoroutine >= sm.before.NumGoroutine {
		d.NumGoroutine = fmt.Sprintf("%d", sm.current.NumGoroutine-sm.before.NumGoroutine)
	} else {
		d.NumGoroutine = fmt.Sprintf("-%d", sm.before.NumGoroutine-sm.current.NumGoroutine)
	}
	return d
}

func (sm *sysMetric) getMem() sysData {
	return sysData{
		Alloc:         size2Human(sm.current.Alloc),
		Lookups:       size2Human(sm.current.Lookups),
		Sys:           size2Human(sm.current.Sys),
		AllocTotal:    size2Human(sm.current.TotalAlloc),
		HeapAlloc:     size2Human(sm.current.HeapAlloc),
		HeapSys:       size2Human(sm.current.HeapSys),
		StackInuse:    size2Human(sm.current.StackInuse),
		StackSys:      size2Human(sm.current.StackSys),
		PauseTotalNs:  sm.current.PauseTotalNs,
		Mallocs:       sm.current.Mallocs,
		Frees:         sm.current.Frees,
		NumGC:         sm.current.NumGC,
		GCCPUFraction: fmt.Sprintf("%.2f%%", sm.current.GCCPUFraction*100),
		NumGoroutine:  fmt.Sprintf("%d", sm.current.NumGoroutine),
	}
}

func size2Human(size uint64) string {
	if size < 1024 {
		return fmt.Sprintf("%dB", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.2fK", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.2fM", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.2fG", float64(size)/(1024*1024*1024))
}
