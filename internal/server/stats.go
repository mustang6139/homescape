package server

import (
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// Stats is a snapshot of host metrics for the system-stats widget.
type Stats struct {
	CPUPercent  float64 `json:"cpuPercent"`
	MemUsed     uint64  `json:"memUsed"`
	MemTotal    uint64  `json:"memTotal"`
	DiskUsed    uint64  `json:"diskUsed"`
	DiskTotal   uint64  `json:"diskTotal"`
	UptimeSecs  uint64  `json:"uptimeSecs"`
	CollectedAt string  `json:"collectedAt"`
}

func (s *Server) handleStats(w http.ResponseWriter, _ *http.Request) {
	st := Stats{CollectedAt: time.Now().UTC().Format(time.RFC3339)}

	// Non-blocking CPU sample (percentage since last call); ignore errors per-metric so a
	// single unavailable source doesn't blank the whole widget.
	if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
		st.CPUPercent = round1(pcts[0])
	}
	if vm, err := mem.VirtualMemory(); err == nil {
		st.MemUsed, st.MemTotal = vm.Used, vm.Total
	}
	if du, err := disk.Usage("/"); err == nil {
		st.DiskUsed, st.DiskTotal = du.Used, du.Total
	}
	if up, err := host.Uptime(); err == nil {
		st.UptimeSecs = up
	}

	writeJSON(w, http.StatusOK, st)
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
