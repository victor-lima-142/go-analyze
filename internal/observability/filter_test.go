package observability

import "testing"

func TestDefaultFilter_IsObservability(t *testing.T) {
	f := DefaultFilter()

	cases := []struct {
		ns, pod, container string
		want               bool
	}{
		{"heavy", "complexcalc-abc", "complexcalc", false},
		{"default", "app", "container", false},
		{"kube-system", "any", "any", true},
		{"kubernetes-dashboard", "x", "x", true},
		{"heavy", "prometheus-server-1", "prometheus-server", true},
		{"heavy", "any", "alertmanager", true},
		{"heavy", "any", "kube-state-metrics", true},
	}
	for _, c := range cases {
		got := f.IsObservability(c.ns, c.pod, c.container)
		if got != c.want {
			t.Errorf("IsObservability(%q,%q,%q) = %v; want %v", c.ns, c.pod, c.container, got, c.want)
		}
	}
}

func TestCustomFilter(t *testing.T) {
	f := NewFilter([]string{"monitoring"}, []string{"datadog"})
	if !f.IsObservability("monitoring", "x", "y") {
		t.Error("custom namespace should match")
	}
	if !f.IsObservability("default", "datadog-agent", "agent") {
		t.Error("custom pattern should match")
	}
	if f.IsObservability("default", "myapp", "mycontainer") {
		t.Error("unrelated workload must not match")
	}
}

func TestNewFilter_EmptyFallsBackToDefault(t *testing.T) {
	f := NewFilter(nil, nil)
	if !f.IsObservability("kube-system", "x", "x") {
		t.Error("expected default namespace")
	}
}

func TestMatchesLabelValues(t *testing.T) {
	f := DefaultFilter()
	if !f.MatchesLabelValues(map[string]string{"namespace": "kube-system"}) {
		t.Error("expected ns match")
	}
	if !f.MatchesLabelValues(map[string]string{"pod": "prometheus-x"}) {
		t.Error("expected pattern match in any label value")
	}
	if f.MatchesLabelValues(map[string]string{"pod": "complexcalc"}) {
		t.Error("benign workload must not match")
	}
}
