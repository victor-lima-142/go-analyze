package metrics

import (
	"strings"
	"testing"
)

func TestNormalizeWindow(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "10m"},
		{"30s", "30s"},
		{"1m", "1m"},
		{"5m", "5m"},
		{"10m", "10m"},
		{"30m", "30m"},
		{"1h", "1h"},
		{"invalid", "10m"},
		{"24h", "10m"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeWindow(tc.input)
			if got != tc.expected {
				t.Errorf("NormalizeWindow(%q) = %q; want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestCPUQueries(t *testing.T) {
	p := QueryParams{
		Namespace: "default",
		Pod:       "nginx-pod",
		Container: "nginx-container",
		Window:    "5m",
	}

	t.Run("CPURequestQuery", func(t *testing.T) {
		q := CPURequestQuery(p)
		if !strings.Contains(q, "kube_pod_container_resource_requests") {
			t.Errorf("expected CPURequestQuery to query requests, got: %s", q)
		}
		if !strings.Contains(q, `namespace="default"`) || !strings.Contains(q, `pod=~"^nginx-pod.*$"`) || !strings.Contains(q, `container="nginx-container"`) {
			t.Errorf("expected CPURequestQuery to contain label matchers, got: %s", q)
		}
	})

	t.Run("CPULimitQuery", func(t *testing.T) {
		q := CPULimitQuery(p)
		if !strings.Contains(q, "kube_pod_container_resource_limits") {
			t.Errorf("expected CPULimitQuery to query limits, got: %s", q)
		}
		if !strings.Contains(q, `namespace="default"`) {
			t.Errorf("expected CPULimitQuery to contain namespace matcher, got: %s", q)
		}
	})

	t.Run("CPUUsageQuery", func(t *testing.T) {
		q := CPUUsageQuery(p)
		if !strings.Contains(q, "container_cpu_usage_seconds_total") {
			t.Errorf("expected CPUUsageQuery to query usage seconds total, got: %s", q)
		}
		if !strings.Contains(q, "[5m]") {
			t.Errorf("expected CPUUsageQuery to contain window duration [5m], got: %s", q)
		}
	})
}

func TestMemoryQueries(t *testing.T) {
	p := QueryParams{
		Namespace: "test-ns",
	}

	t.Run("MemoryRequestQuery without pod/container", func(t *testing.T) {
		q := MemoryRequestQuery(p)
		if !strings.Contains(q, `namespace="test-ns"`) {
			t.Errorf("expected MemoryRequestQuery to contain namespace matcher, got: %s", q)
		}
		if !strings.Contains(q, `container!="POD"`) {
			t.Errorf("expected MemoryRequestQuery to filter out container=\"POD\" when container is empty, got: %s", q)
		}
	})

	t.Run("MemoryLimitQuery", func(t *testing.T) {
		q := MemoryLimitQuery(p)
		if !strings.Contains(q, "kube_pod_container_resource_limits") {
			t.Errorf("expected MemoryLimitQuery to query limits, got: %s", q)
		}
	})

	t.Run("MemoryUsageQuery", func(t *testing.T) {
		q := MemoryUsageQuery(p)
		if !strings.Contains(q, "container_memory_working_set_bytes") {
			t.Errorf("expected MemoryUsageQuery to query container_memory_working_set_bytes, got: %s", q)
		}
	})
}
