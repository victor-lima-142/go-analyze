"""Exact workload-scoped collection from the analyzer API and Prometheus text."""
import re
from urllib.parse import urlencode
from . import config


def read_indicator(base_url, namespace, pod, container, window="5m"):
    query = urlencode({"namespace": namespace, "pod": pod, "container": container, "window": window})
    data = config.http_get_json(f"{base_url}/api/v1/indicators?{query}")
    for item in data.get("content", data.get("items", [])) or []:
        if (item.get("namespace"), item.get("pod"), item.get("container")) == (namespace, pod, container):
            return item
    return None


def parse_metric(text: str, metric: str, labels: dict) -> float:
    total = 0.0
    found = False
    for line in text.splitlines():
        if line.startswith("#") or not line.startswith(metric + "{"):
            continue
        match = re.match(r'[^\{]+\{([^}]*)\}\s+([0-9eE.+-]+)$', line)
        if not match:
            continue
        actual = dict(re.findall(r'(\w+)="((?:\\.|[^"])*)"', match.group(1)))
        if all(actual.get(k) == v for k, v in labels.items()):
            total += float(match.group(2)); found = True
    return total if found else 0.0


def read_notification_counter(metrics_url, metric, indicator, namespace, pod, container):
    labels = {"indicator": indicator, "scope": "workload", "namespace": namespace, "pod": pod, "container": container}
    return parse_metric(config.http_get_text(metrics_url), metric, labels)


def read_sustained_duration(metrics_url, indicator, namespace, pod, container):
    labels = {"indicator": indicator, "scope": "workload", "namespace": namespace, "pod": pod, "container": container}
    return parse_metric(config.http_get_text(metrics_url), "go_analyze_notification_sustained_duration_seconds", labels)


def audit_cost(base_url, period="30min"):
    return config.http_get_json(f"{base_url}/api/v1/audit?period={period}")


def read_consolidated(base_url, page_size=1000):
    return config.http_get_json(f"{base_url}/api/v1/consolidated?pageSize={page_size}")


def read_workload_details(base_url, namespace, pod, container, window="5m"):
    query = urlencode({"namespace": namespace, "pod": pod, "container": container, "window": window, "limit": 1000})
    return config.http_get_json(f"{base_url}/api/v1/workloads/details?{query}")


def prometheus_query(prometheus_url, query):
    data = config.http_get_json(f"{prometheus_url}/api/v1/query?{urlencode({'query': query})}")
    if data.get("status") != "success":
        raise RuntimeError(f"Prometheus query failed: {data}")
    return data.get("data", {}).get("result", [])
