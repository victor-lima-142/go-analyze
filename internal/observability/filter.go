package observability

import "strings"

type Filter struct {
	Namespaces []string
	Patterns   []string
}

func DefaultFilter() *Filter {
	return &Filter{
		Namespaces: []string{"kube-system", "kubernetes-dashboard"},
		Patterns: []string{
			"prometheus",
			"kube-state-metrics",
			"node-exporter",
			"pushgateway",
			"configmap-reload",
			"alertmanager",
		},
	}
}

func NewFilter(namespaces, patterns []string) *Filter {
	f := &Filter{Namespaces: namespaces, Patterns: patterns}
	if len(f.Namespaces) == 0 {
		f.Namespaces = DefaultFilter().Namespaces
	}
	if len(f.Patterns) == 0 {
		f.Patterns = DefaultFilter().Patterns
	}
	return f
}

func (f *Filter) IsObservability(namespace, pod, container string) bool {
	for _, ns := range f.Namespaces {
		if namespace == ns {
			return true
		}
	}
	podLower := strings.ToLower(pod)
	containerLower := strings.ToLower(container)
	for _, p := range f.Patterns {
		p = strings.ToLower(p)
		if p == "" {
			continue
		}
		if strings.Contains(podLower, p) || strings.Contains(containerLower, p) {
			return true
		}
	}
	return false
}

func (f *Filter) MatchesLabelValues(labels map[string]string) bool {
	for _, ns := range f.Namespaces {
		if labels["namespace"] == ns {
			return true
		}
	}
	for _, p := range f.Patterns {
		pLower := strings.ToLower(p)
		if pLower == "" {
			continue
		}
		for _, val := range labels {
			if strings.Contains(strings.ToLower(val), pLower) {
				return true
			}
		}
	}
	return false
}
