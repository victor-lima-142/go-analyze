package metrics

import (
	"context"
	"math"
	"strings"
	"sync"
	"testing"

	"github.com/victor-lima-142/go-analyze/internal/observability"
)

type calculatorClient struct {
	mu      sync.Mutex
	queries []string
}

func (c *calculatorClient) QueryInstant(_ context.Context, query string) ([]Sample, error) {
	c.mu.Lock()
	c.queries = append(c.queries, query)
	c.mu.Unlock()
	sample := func(pod, container string, value float64) Sample {
		return Sample{Labels: map[string]string{"namespace": "n", "pod": pod, "container": container}, Value: value}
	}
	switch {
	case strings.Contains(query, "resource_requests") && strings.Contains(query, `resource="cpu"`):
		return []Sample{sample("z", "app", 1), sample("a", "app", 2)}, nil
	case strings.Contains(query, "resource_limits") && strings.Contains(query, `resource="cpu"`):
		return []Sample{sample("z", "app", 2), sample("a", "app", 3)}, nil
	case strings.Contains(query, "cpu_usage_seconds"):
		return []Sample{sample("z", "app", .5), sample("a", "app", 1)}, nil
	case strings.Contains(query, "resource_requests") && strings.Contains(query, `resource="memory"`):
		return []Sample{sample("z", "app", 2*1024*1024*1024), sample("a", "app", 3*1024*1024*1024)}, nil
	case strings.Contains(query, "resource_limits") && strings.Contains(query, `resource="memory"`):
		return []Sample{sample("z", "app", 3*1024*1024*1024), sample("a", "app", 4*1024*1024*1024)}, nil
	case strings.Contains(query, "memory_working_set"):
		return []Sample{sample("z", "app", 1*1024*1024*1024), sample("a", "app", 2*1024*1024*1024)}, nil
	default:
		return nil, nil
	}
}

func newTestCalculator() *Calculator {
	return NewCalculator(CalculatorOptions{
		Filter: observability.DefaultFilter(),
	})
}

func TestIndexByWorkload_FiltersObservability(t *testing.T) {
	calc := newTestCalculator()
	samples := []Sample{
		{
			Labels: map[string]string{"namespace": "heavy", "pod": "complexcalc-12345", "container": "complexcalc"},
			Value:  0.5,
		},
		{
			Labels: map[string]string{"namespace": "heavy", "pod": "prometheus-server-123", "container": "prometheus-server"},
			Value:  1.2,
		},
	}

	indexed := calc.indexByWorkload(samples)

	if _, ok := indexed["heavy/complexcalc-12345/complexcalc"]; !ok {
		t.Error("Expected to find complexcalc workload")
	}
	if _, ok := indexed["heavy/prometheus-server-123/prometheus-server"]; ok {
		t.Error("Expected prometheus workload to be filtered out")
	}
}

func TestSumValues_FiltersByLabel(t *testing.T) {
	calc := newTestCalculator()
	samples := []Sample{
		{Labels: map[string]string{"namespace": "heavy", "pod": "complexcalc-123"}, Value: 10},
		{Labels: map[string]string{"namespace": "heavy", "pod": "prometheus-server-123"}, Value: 20},
		{Labels: map[string]string{"namespace": "kube-system", "pod": "kube-dns"}, Value: 30},
	}

	sum := calc.sumValues(samples)
	if sum != 10 {
		t.Errorf("Expected sum=10 (filtered), got %f", sum)
	}
}

func TestMergeUsageProportional_NoDuplication(t *testing.T) {
	useMap := map[string]float64{
		"heavy/pod-a/": 12.0,
	}
	reqMap := map[string]float64{
		"heavy/pod-a/c1": 1.0,
		"heavy/pod-a/c2": 3.0,
	}
	limMap := map[string]float64{}
	mergeUsageProportional(useMap, reqMap, limMap)

	if _, ok := useMap["heavy/pod-a/"]; ok {
		t.Error("expected empty-container key to be removed after redistribution")
	}
	if v := useMap["heavy/pod-a/c1"]; v != 3.0 {
		t.Errorf("expected c1 to get 25%% (3.0) of usage, got %f", v)
	}
	if v := useMap["heavy/pod-a/c2"]; v != 9.0 {
		t.Errorf("expected c2 to get 75%% (9.0) of usage, got %f", v)
	}
}

func TestMergeUsageProportional_FallsBackToLimits(t *testing.T) {
	useMap := map[string]float64{
		"heavy/pod-a/": 10.0,
	}
	reqMap := map[string]float64{}
	limMap := map[string]float64{
		"heavy/pod-a/c1": 1.0,
		"heavy/pod-a/c2": 1.0,
	}
	mergeUsageProportional(useMap, reqMap, limMap)

	if useMap["heavy/pod-a/c1"] != 5.0 || useMap["heavy/pod-a/c2"] != 5.0 {
		t.Errorf("expected equal split via limits, got %+v", useMap)
	}
}

func TestCalculatorRatios(t *testing.T) {
	if r := CPUWasteRatio(1, 0.4); r != 0.6 {
		t.Errorf("CPUWasteRatio expected 0.6, got %f", r)
	}
	if r := MemWasteRatio(0, 0); r != 0 {
		t.Errorf("MemWasteRatio with zero req expected 0, got %f", r)
	}
	if r := CPUWasteRatio(1, 2); r != 0 {
		t.Errorf("CPUWasteRatio must clamp negative values, got %f", r)
	}
	if r := MemWasteRatio(1, -1); r != 1 {
		t.Errorf("MemWasteRatio must clamp values above one, got %f", r)
	}
}

func TestProjectedMonthlyWasteUSD(t *testing.T) {
	cost := ProjectedMonthlyWasteUSD(2, 1, 2*1024*1024*1024, 1*1024*1024*1024, 0.04048, 0.004445, 720)
	expected := 0.04048*1*720 + 0.004445*1*720
	if cost < expected-0.01 || cost > expected+0.01 {
		t.Errorf("ProjectedMonthlyWasteUSD = %f; expected ≈ %f", cost, expected)
	}
	if cpu, memory := CPUProjectedMonthlyWasteUSD(2, 1, 0.04048, 720), MemoryProjectedMonthlyWasteUSD(2*1024*1024*1024, 1*1024*1024*1024, 0.004445, 720); math.Abs(cost-(cpu+memory)) > 1e-9 {
		t.Errorf("total %f must equal cpu %f + memory %f", cost, cpu, memory)
	}
}

func TestCalculatorCalculateSixQueriesSortedAndDecomposed(t *testing.T) {
	client := &calculatorClient{}
	calculator := NewCalculator(CalculatorOptions{Client: client, CPUHourlyUSD: 1, MemoryGiBHourlyUSD: 2, MonthlyHours: 1})
	result, err := calculator.Calculate(context.Background(), QueryParams{Window: "5m"})
	if err != nil {
		t.Fatal(err)
	}
	if len(client.queries) != 6 {
		t.Fatalf("queries=%d want=6", len(client.queries))
	}
	if len(result.Items) != 2 || result.Items[0].Pod != "a" || result.Items[1].Pod != "z" {
		t.Fatalf("items not sorted: %+v", result.Items)
	}
	if result.Items[0].CPURequestedCores != 2 || result.Items[0].MemoryRequestedBytes != 3*1024*1024*1024 {
		t.Fatalf("distinct series were not joined correctly: %+v", result.Items[0])
	}
	for _, item := range result.Items {
		if math.Abs(item.ProjectedMonthlyWasteUSD-(item.CPUProjectedMonthlyWasteUSD+item.MemoryProjectedMonthlyWasteUSD)) > 1e-9 {
			t.Fatalf("item costs do not sum: %+v", item)
		}
	}
	if math.Abs(result.Indicators.ProjectedMonthlyWasteUSD-(result.Indicators.CPUProjectedMonthlyWasteUSD+result.Indicators.MemoryProjectedMonthlyWasteUSD)) > 1e-9 {
		t.Fatalf("aggregate costs do not sum")
	}
}
