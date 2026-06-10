package metrics

import (
	"strings"
	"testing"
	"time"
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

func TestFormatDurationPromQL(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{72 * time.Hour, "3d"},
		{7 * 24 * time.Hour, "7d"},
		{2 * time.Hour, "2h"},
		{30 * time.Minute, "30m"},
		{0, "3d"},
	}
	for _, c := range cases {
		got := FormatDurationPromQL(c.d)
		if got != c.want {
			t.Errorf("FormatDurationPromQL(%s) = %q; want %q", c.d, got, c.want)
		}
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
		if !strings.Contains(q, `namespace="default"`) || !strings.Contains(q, `pod="nginx-pod"`) || !strings.Contains(q, `container="nginx-container"`) {
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

func TestHPAAndPVCQueries(t *testing.T) {
	p := QueryParams{
		Namespace: "prod-ns",
		HPA:       "my-hpa",
		PVC:       "my-pvc",
	}

	t.Run("HPAAvgReplicasQuery", func(t *testing.T) {
		q := HPAAvgReplicasQuery(p, 72*time.Hour)
		if !strings.Contains(q, "kube_horizontalpodautoscaler_status_current_replicas") {
			t.Errorf("expected HPAAvgReplicasQuery to query replicas, got: %s", q)
		}
		if !strings.Contains(q, `namespace="prod-ns"`) || !strings.Contains(q, `horizontalpodautoscaler="my-hpa"`) {
			t.Errorf("expected HPAAvgReplicasQuery to contain namespace and hpa matchers, got: %s", q)
		}
		if !strings.Contains(q, "[3d]") {
			t.Errorf("expected HPAAvgReplicasQuery to contain configured [3d] window, got: %s", q)
		}
	})

	t.Run("HPAMaxReplicasQuery", func(t *testing.T) {
		q := HPAMaxReplicasQuery(p, 72*time.Hour)
		if !strings.Contains(q, "kube_horizontalpodautoscaler_spec_max_replicas") {
			t.Errorf("expected HPAMaxReplicasQuery to query spec max replicas, got: %s", q)
		}
	})

	t.Run("PVCCapacityQuery", func(t *testing.T) {
		q := PVCCapacityQuery(p)
		if !strings.Contains(q, "kubelet_volume_stats_capacity_bytes") {
			t.Errorf("expected PVCCapacityQuery to query volume capacity, got: %s", q)
		}
		if !strings.Contains(q, `namespace="prod-ns"`) || !strings.Contains(q, `persistentvolumeclaim="my-pvc"`) {
			t.Errorf("expected PVCCapacityQuery to contain namespace and pvc matchers, got: %s", q)
		}
	})

	t.Run("PVCUsedQuery", func(t *testing.T) {
		q := PVCUsedQuery(p)
		if !strings.Contains(q, "kubelet_volume_stats_used_bytes") {
			t.Errorf("expected PVCUsedQuery to query volume usage, got: %s", q)
		}
	})
}
