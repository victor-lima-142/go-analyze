package metrics

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"sort"
	"sync"

	"github.com/victor-lima-142/go-analyze/internal/numeric"
	"github.com/victor-lima-142/go-analyze/internal/observability"
)

type Calculator struct {
	client             PrometheusClient
	cpuHourlyUSD       float64
	memoryGiBHourlyUSD float64
	monthlyHours       float64
	costModelLabel     string
	logger             *slog.Logger
	filter             *observability.Filter
}

func NewCalculator(opts CalculatorOptions) *Calculator {
	filter := opts.Filter
	if filter == nil {
		filter = observability.DefaultFilter()
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Calculator{
		client:             opts.Client,
		cpuHourlyUSD:       opts.CPUHourlyUSD,
		memoryGiBHourlyUSD: opts.MemoryGiBHourlyUSD,
		monthlyHours:       opts.MonthlyHours,
		costModelLabel:     opts.CostModelLabel,
		logger:             logger,
		filter:             filter,
	}
}

type instantResult struct {
	samples []Sample
	err     error
}

func (c *Calculator) parallelInstant(ctx context.Context, queries map[string]string) map[string]instantResult {
	results := make(map[string]instantResult, len(queries))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for name, query := range queries {
		wg.Add(1)
		go func(name, query string) {
			defer wg.Done()
			samples, err := c.client.QueryInstant(ctx, query)
			mu.Lock()
			results[name] = instantResult{samples: samples, err: err}
			mu.Unlock()
		}(name, query)
	}
	wg.Wait()
	return results
}

func (c *Calculator) Calculate(ctx context.Context, p QueryParams) (*IndicatorResponse, error) {
	p.Window = NormalizeWindow(p.Window)

	core := map[string]string{
		"cpuReq": CPURequestQuery(p),
		"cpuUse": CPUUsageQuery(p),
		"cpuLim": CPULimitQuery(p),
		"memReq": MemoryRequestQuery(p),
		"memUse": MemoryUsageQuery(p),
		"memLim": MemoryLimitQuery(p),
	}
	results := c.parallelInstant(ctx, core)

	for _, key := range []string{"cpuReq", "cpuUse", "cpuLim", "memReq", "memUse", "memLim"} {
		if r := results[key]; r.err != nil {
			return nil, errors.Join(r.err)
		}
	}

	byCPUReq := c.indexByWorkload(results["cpuReq"].samples)
	byCPUUse := c.indexByWorkload(results["cpuUse"].samples)
	byCPULim := c.indexByWorkload(results["cpuLim"].samples)
	byMemReq := c.indexByWorkload(results["memReq"].samples)
	byMemUse := c.indexByWorkload(results["memUse"].samples)
	byMemLim := c.indexByWorkload(results["memLim"].samples)

	mergeUsageProportional(byCPUUse, byCPUReq, byCPULim)
	mergeUsageProportional(byMemUse, byMemReq, byMemLim)

	keys := unionKeys(byCPUReq, byCPUUse, byMemReq, byMemUse)
	items := make([]WorkloadItem, 0, len(keys))

	var totalCPURequested, totalCPUUsed, totalCPULimit float64
	var totalMemRequested, totalMemUsed, totalMemLimit float64
	var totalCPUProjectedWaste, totalMemoryProjectedWaste float64

	for _, key := range keys {
		cpuRequested := byCPUReq[key]
		cpuUsed := byCPUUse[key]
		cpuLimit := byCPULim[key]
		memRequested := byMemReq[key]
		memUsed := byMemUse[key]
		memLimit := byMemLim[key]

		totalCPURequested += cpuRequested
		totalCPUUsed += cpuUsed
		totalCPULimit += cpuLimit
		totalMemRequested += memRequested
		totalMemUsed += memUsed
		totalMemLimit += memLimit

		cpuWasteRatio := CPUWasteRatio(cpuRequested, cpuUsed)
		memWasteRatio := MemWasteRatio(memRequested, memUsed)
		cpuProjected := CPUProjectedMonthlyWasteUSD(cpuRequested, cpuUsed, c.cpuHourlyUSD, c.monthlyHours)
		memoryProjected := MemoryProjectedMonthlyWasteUSD(memRequested, memUsed, c.memoryGiBHourlyUSD, c.monthlyHours)
		projected := numeric.Round4(cpuProjected + memoryProjected)
		totalCPUProjectedWaste += cpuProjected
		totalMemoryProjectedWaste += memoryProjected

		labels := splitKey(key)
		items = append(items, WorkloadItem{
			Namespace:                      labels[0],
			Pod:                            labels[1],
			Container:                      labels[2],
			CPURequestedCores:              cpuRequested,
			CPUUsedCores:                   cpuUsed,
			CPULimitCores:                  cpuLimit,
			MemoryRequestedBytes:           memRequested,
			MemoryUsedBytes:                memUsed,
			MemoryLimitBytes:               memLimit,
			CPUWasteRatio:                  cpuWasteRatio,
			MemWasteRatio:                  memWasteRatio,
			CPUProjectedMonthlyWasteUSD:    cpuProjected,
			MemoryProjectedMonthlyWasteUSD: memoryProjected,
			ProjectedMonthlyWasteUSD:       projected,
		})
	}

	indicators := Indicators{
		CPUWasteRatio:                  CPUWasteRatio(totalCPURequested, totalCPUUsed),
		MemWasteRatio:                  MemWasteRatio(totalMemRequested, totalMemUsed),
		CPUProjectedMonthlyWasteUSD:    numeric.Round4(totalCPUProjectedWaste),
		MemoryProjectedMonthlyWasteUSD: numeric.Round4(totalMemoryProjectedWaste),
		ProjectedMonthlyWasteUSD:       numeric.Round4(totalCPUProjectedWaste + totalMemoryProjectedWaste),
	}

	return &IndicatorResponse{
		Filters:    p,
		Indicators: indicators,
		Inputs: Inputs{
			CPURequestedCores:    totalCPURequested,
			CPUUsedCores:         totalCPUUsed,
			CPULimitCores:        totalCPULimit,
			MemoryRequestedBytes: totalMemRequested,
			MemoryUsedBytes:      totalMemUsed,
			MemoryLimitBytes:     totalMemLimit,
		},
		Items:     items,
		CostModel: c.costModelLabel,
	}, nil
}

func CPUWasteRatio(requested, used float64) float64 {
	return wasteRatio(requested, used)
}

func MemWasteRatio(requested, used float64) float64 {
	return wasteRatio(requested, used)
}

func wasteRatio(requested, used float64) float64 {
	if requested <= 0 {
		return 0
	}
	return numeric.Round4(math.Min(1, math.Max(0, (requested-used)/requested)))
}

func CPUProjectedMonthlyWasteUSD(requested, used, hourlyUSD, monthlyHours float64) float64 {
	return numeric.Round4(math.Max(requested-used, 0) * hourlyUSD * monthlyHours)
}

func MemoryProjectedMonthlyWasteUSD(requestedBytes, usedBytes, gibHourlyUSD, monthlyHours float64) float64 {
	wasteGiB := math.Max(requestedBytes-usedBytes, 0) / 1024 / 1024 / 1024
	return numeric.Round4(wasteGiB * gibHourlyUSD * monthlyHours)
}

func ProjectedMonthlyWasteUSD(cpuRequested, cpuUsed, memRequestedBytes, memUsedBytes, cpuHourlyUSD, memGiBHourlyUSD, monthlyHours float64) float64 {
	return numeric.Round4(CPUProjectedMonthlyWasteUSD(cpuRequested, cpuUsed, cpuHourlyUSD, monthlyHours) + MemoryProjectedMonthlyWasteUSD(memRequestedBytes, memUsedBytes, memGiBHourlyUSD, monthlyHours))
}

func (c *Calculator) indexByWorkload(samples []Sample) map[string]float64 {
	out := map[string]float64{}
	for _, s := range samples {
		ns := s.Labels["namespace"]
		pod := s.Labels["pod"]
		container := s.Labels["container"]
		if c.filter.IsObservability(ns, pod, container) {
			continue
		}
		key := ns + "/" + pod + "/" + container
		out[key] += s.Value
	}
	return out
}

// mergeUsageProportional reassigns usage values that arrived with an empty
// container label, distributing them proportionally across containers of the
// same pod that have a request defined. Falls back to limit-based distribution
// when no requests exist. Avoids the previous bug of duplicating the value to
// every matching container.
func mergeUsageProportional(useMap, reqMap, limMap map[string]float64) {
	for key, val := range useMap {
		labels := splitKey(key)
		ns, pod, container := labels[0], labels[1], labels[2]
		if container != "" {
			continue
		}
		// gather candidates with the same (ns, pod) and a defined request/limit
		type candidate struct {
			key    string
			weight float64
		}
		var candidates []candidate
		var totalWeight float64

		gather := func(src map[string]float64) {
			for otherKey, w := range src {
				ol := splitKey(otherKey)
				if ol[0] == ns && ol[1] == pod && ol[2] != "" && w > 0 {
					candidates = append(candidates, candidate{key: otherKey, weight: w})
					totalWeight += w
				}
			}
		}
		gather(reqMap)
		if len(candidates) == 0 {
			gather(limMap)
		}

		if len(candidates) == 0 {
			continue
		}
		for _, cand := range candidates {
			useMap[cand.key] += val * (cand.weight / totalWeight)
		}
		delete(useMap, key)
	}
}

func unionKeys(maps ...map[string]float64) []string {
	seen := map[string]struct{}{}
	for _, m := range maps {
		for key := range m {
			seen[key] = struct{}{}
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := splitKey(keys[i]), splitKey(keys[j])
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		if a[1] != b[1] {
			return a[1] < b[1]
		}
		return a[2] < b[2]
	})
	return keys
}

func splitKey(key string) [3]string {
	var out [3]string
	part := 0
	start := 0
	for i, r := range key {
		if r == '/' && part < 2 {
			out[part] = key[start:i]
			start = i + 1
			part++
		}
	}
	out[part] = key[start:]
	return out
}

func (c *Calculator) sumValues(samples []Sample) float64 {
	var total float64
	for _, s := range samples {
		if c.filter.MatchesLabelValues(s.Labels) {
			continue
		}
		total += s.Value
	}
	return total
}
