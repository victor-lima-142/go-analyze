package metrics

import (
	"testing"

	"go-analyze/internal/observability"
)

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

func TestOOMRiskScore(t *testing.T) {
	if got := OOMRiskScore(0, 0); got != 0 {
		t.Errorf("expected 0 with no limit, got %f", got)
	}
	if got := OOMRiskScore(512, 1024); got != 0.5 {
		t.Errorf("expected 0.5, got %f", got)
	}
	if got := OOMRiskScore(2000, 1000); got != 1.0 {
		t.Errorf("expected clamped to 1.0, got %f", got)
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
	if r := PVCWasteRatio(100, 25); r != 0.75 {
		t.Errorf("PVCWasteRatio expected 0.75, got %f", r)
	}
	if r := HPAEfficiency(1, 4); r != 0.25 {
		t.Errorf("HPAEfficiency expected 0.25, got %f", r)
	}
}

func TestProjectedMonthlyWasteUSD(t *testing.T) {
	cost := ProjectedMonthlyWasteUSD(2, 1, 2*1024*1024*1024, 1*1024*1024*1024, 0.04048, 0.004445, 720)
	expected := 0.04048*1*720 + 0.004445*1*720
	if cost < expected-0.01 || cost > expected+0.01 {
		t.Errorf("ProjectedMonthlyWasteUSD = %f; expected ≈ %f", cost, expected)
	}
}
