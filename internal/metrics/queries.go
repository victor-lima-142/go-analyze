package metrics

import (
	"fmt"
	"math"
	"strings"
	"time"
)

func NormalizeWindow(window string) string {
	switch window {
	case "30s", "1m", "5m", "10m", "30m", "1h":
		return window
	case "":
		return "10m"
	default:
		return "10m"
	}
}

// FormatDurationPromQL formats a Go duration as a PromQL-compatible string
// using the largest sensible unit (d/h/m/s).
func FormatDurationPromQL(d time.Duration) string {
	if d <= 0 {
		return "3d"
	}
	if d%(24*time.Hour) == 0 {
		return fmt.Sprintf("%dd", int(d/(24*time.Hour)))
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%ds", int(math.Round(d.Seconds())))
}

func labelMatchers(p QueryParams, includeContainer bool) string {
	parts := make([]string, 0, 6)
	if p.Namespace != "" {
		parts = append(parts, fmt.Sprintf("namespace=%q", p.Namespace))
	}
	if p.Pod != "" {
		parts = append(parts, fmt.Sprintf("pod=%q", p.Pod))
	}
	if includeContainer {
		if p.Container != "" {
			parts = append(parts, fmt.Sprintf("container=%q", p.Container))
		} else {
			parts = append(parts, `container!="POD"`)
		}
	}
	return strings.Join(parts, ",")
}

func CPURequestQuery(p QueryParams) string {
	return fmt.Sprintf("sum(kube_pod_container_resource_requests{resource=\"cpu\",%s}) by (namespace,pod,container)", labelMatchers(p, true))
}

func CPULimitQuery(p QueryParams) string {
	return fmt.Sprintf("sum(kube_pod_container_resource_limits{resource=\"cpu\",%s}) by (namespace,pod,container)", labelMatchers(p, true))
}

func CPUUsageQuery(p QueryParams) string {
	p.Window = NormalizeWindow(p.Window)
	return fmt.Sprintf("sum(rate(container_cpu_usage_seconds_total{%s}[%s])) by (namespace,pod,container)", labelMatchers(p, true), p.Window)
}

func MemoryRequestQuery(p QueryParams) string {
	return fmt.Sprintf("sum(kube_pod_container_resource_requests{resource=\"memory\",%s}) by (namespace,pod,container)", labelMatchers(p, true))
}

func MemoryLimitQuery(p QueryParams) string {
	return fmt.Sprintf("sum(kube_pod_container_resource_limits{resource=\"memory\",%s}) by (namespace,pod,container)", labelMatchers(p, true))
}

func MemoryUsageQuery(p QueryParams) string {
	return fmt.Sprintf("sum(container_memory_working_set_bytes{%s}) by (namespace,pod,container)", labelMatchers(p, true))
}

func HPAAvgReplicasQuery(p QueryParams, window time.Duration) string {
	matchers := []string{}
	if p.Namespace != "" {
		matchers = append(matchers, fmt.Sprintf("namespace=%q", p.Namespace))
	}
	if p.HPA != "" {
		matchers = append(matchers, fmt.Sprintf("horizontalpodautoscaler=%q", p.HPA))
	}
	return fmt.Sprintf("avg(avg_over_time(kube_horizontalpodautoscaler_status_current_replicas{%s}[%s])) by (namespace,horizontalpodautoscaler)", strings.Join(matchers, ","), FormatDurationPromQL(window))
}

func HPAMaxReplicasQuery(p QueryParams, window time.Duration) string {
	matchers := []string{}
	if p.Namespace != "" {
		matchers = append(matchers, fmt.Sprintf("namespace=%q", p.Namespace))
	}
	if p.HPA != "" {
		matchers = append(matchers, fmt.Sprintf("horizontalpodautoscaler=%q", p.HPA))
	}
	return fmt.Sprintf("max(max_over_time(kube_horizontalpodautoscaler_spec_max_replicas{%s}[%s])) by (namespace,horizontalpodautoscaler)", strings.Join(matchers, ","), FormatDurationPromQL(window))
}

func PVCCapacityQuery(p QueryParams) string {
	matchers := pvcMatchers(p)
	return fmt.Sprintf("sum(kubelet_volume_stats_capacity_bytes{%s}) by (namespace,persistentvolumeclaim)", matchers)
}

func PVCUsedQuery(p QueryParams) string {
	matchers := pvcMatchers(p)
	return fmt.Sprintf("sum(kubelet_volume_stats_used_bytes{%s}) by (namespace,persistentvolumeclaim)", matchers)
}

func pvcMatchers(p QueryParams) string {
	parts := []string{}
	if p.Namespace != "" {
		parts = append(parts, fmt.Sprintf("namespace=%q", p.Namespace))
	}
	if p.PVC != "" {
		parts = append(parts, fmt.Sprintf("persistentvolumeclaim=%q", p.PVC))
	}
	return strings.Join(parts, ",")
}
