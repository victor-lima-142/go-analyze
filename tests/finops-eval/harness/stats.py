"""Classification and confidence intervals for experiment results."""
import math


def classify_run(label: str, notified: bool, status: str = "valid") -> str:
    if status != "valid":
        return status
    if label == "positive":
        return "TP" if notified else "FN"
    return "FP" if notified else "TN"


def wilson95(successes: int, total: int):
    if total == 0:
        return None, None
    z = 1.959963984540054
    p = successes / total
    den = 1 + z * z / total
    center = (p + z * z / (2 * total)) / den
    half = z * math.sqrt((p * (1 - p) + z * z / (4 * total)) / total) / den
    return round(max(0, center - half), 4), round(min(1, center + half), 4)


def aggregate_scenario(name: str, label: str, runs: list) -> dict:
    valid = [r for r in runs if r.get("status") == "valid"]
    counts = {key: 0 for key in ("TP", "FN", "FP", "TN")}
    for run in valid:
        counts[classify_run(label, bool(run.get("notified")))] += 1
    positives = counts["TP"] + counts["FN"]
    negatives = counts["FP"] + counts["TN"]
    rate_n = positives if label == "positive" else negatives
    successes = counts["TP"] if label == "positive" else counts["FP"]
    low, high = wilson95(successes, rate_n)
    return {
        "scenario": name, "label": label, **counts,
        "runs": len(runs), "valid_runs": len(valid),
        "invalid_runs": sum(r.get("status") == "invalid" for r in runs),
        "error_runs": sum(r.get("status") == "error" for r in runs),
        "detection_rate": round(counts["TP"] / positives, 4) if positives else None,
        "false_positive_rate": round(counts["FP"] / negatives, 4) if negatives else None,
        "rate_wilson95_low": low, "rate_wilson95_high": high,
    }
