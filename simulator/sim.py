import os
import random
import threading
import time
from datetime import datetime, timezone

import requests

API_URL = os.getenv("API_URL", "http://api:8080/api/measurements")
DEVICE_ID = os.getenv("DEVICE_ID", "sensor-1")
INTERVAL_SEC = float(os.getenv("INTERVAL_SEC", "2"))
REQUEST_TIMEOUT_SEC = float(os.getenv("REQUEST_TIMEOUT_SEC", "3"))
WORKERS = int(os.getenv("WORKERS", "1"))
REPORT_EVERY_SEC = float(os.getenv("REPORT_EVERY_SEC", "10"))


class Stats:
    def __init__(self) -> None:
        self.lock = threading.Lock()
        self.ok = 0
        self.errors = 0
        self.total_latency_ms = 0.0

    def add(self, ok: bool, latency_ms: float) -> None:
        with self.lock:
            if ok:
                self.ok += 1
            else:
                self.errors += 1
            self.total_latency_ms += latency_ms

    def snapshot_and_reset(self) -> tuple[int, int, float]:
        with self.lock:
            ok, errors, total_latency_ms = self.ok, self.errors, self.total_latency_ms
            self.ok = 0
            self.errors = 0
            self.total_latency_ms = 0.0
            return ok, errors, total_latency_ms


def gen_measurement(worker_id: int) -> dict:
    temperature = round(random.uniform(5.0, 25.0), 2)
    ph = round(random.uniform(6.5, 8.5), 2)

    return {
        "deviceId": f"{DEVICE_ID}-w{worker_id}" if WORKERS > 1 else DEVICE_ID,
        "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "temperature": temperature,
        "ph": ph,
    }


def worker(worker_id: int, stats: Stats) -> None:
    while True:
        payload = gen_measurement(worker_id)
        started = time.perf_counter()
        try:
            response = requests.post(API_URL, json=payload, timeout=REQUEST_TIMEOUT_SEC)
            latency_ms = (time.perf_counter() - started) * 1000
            ok = 200 <= response.status_code < 300
            stats.add(ok, latency_ms)
            print(
                f"[sim-{worker_id}] -> {response.status_code} "
                f"latency={latency_ms:.1f}ms payload={payload}"
            )
        except Exception as exc:
            latency_ms = (time.perf_counter() - started) * 1000
            stats.add(False, latency_ms)
            print(f"[sim-{worker_id}] error: {exc}")

        time.sleep(INTERVAL_SEC)


def reporter(stats: Stats) -> None:
    while True:
        time.sleep(REPORT_EVERY_SEC)
        ok, errors, total_latency_ms = stats.snapshot_and_reset()
        total = ok + errors
        if total == 0:
            continue
        avg_latency = total_latency_ms / total
        rps = total / REPORT_EVERY_SEC
        err_rate = (errors / total) * 100
        print(
            f"[report] total={total} ok={ok} errors={errors} "
            f"errorRate={err_rate:.2f}% avgLatency={avg_latency:.1f}ms rps={rps:.2f}"
        )


def main() -> None:
    print(
        f"[sim] sending to {API_URL}, device={DEVICE_ID}, "
        f"workers={WORKERS}, interval={INTERVAL_SEC}s"
    )

    stats = Stats()
    threads = []
    for worker_id in range(1, WORKERS + 1):
        t = threading.Thread(target=worker, args=(worker_id, stats), daemon=True)
        t.start()
        threads.append(t)

    reporter(stats)


if __name__ == "__main__":
    main()
