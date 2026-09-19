package metrics

import (
	"fmt"
	"regexp"
	"strings"
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
func labelMatchers(p QueryParams, includeContainer bool) string {
	parts := make([]string, 0, 6)
	if p.Namespace != "" {
		parts = append(parts, fmt.Sprintf("namespace=%q", p.Namespace))
	}
	if p.Pod != "" {
		pattern := regexp.QuoteMeta(p.Pod)
		pattern = strings.ReplaceAll(pattern, "%", ".*")
		if !strings.Contains(p.Pod, "%") {
			pattern += ".*"
		}
		parts = append(parts, fmt.Sprintf("pod=~%q", "^"+pattern+"$"))
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
