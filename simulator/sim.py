import os
import random
import time
from datetime import datetime, timezone

import requests

API_URL = os.getenv("API_URL", "http://api:8080/api/measurements")
DEVICE_ID = os.getenv("DEVICE_ID", "sensor-1")
INTERVAL_SEC = float(os.getenv("INTERVAL_SEC", "2"))


def gen_measurement() -> dict:
    temperature = round(random.uniform(5.0, 25.0), 2)
    ph = round(random.uniform(6.5, 8.5), 2)

    return {
        "deviceId": DEVICE_ID,
        "timestamp": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
        "temperature": temperature,
        "ph": ph,
    }


def main() -> None:
    print(f"[sim] sending to {API_URL}, device={DEVICE_ID}, interval={INTERVAL_SEC}s")
    while True:
        payload = gen_measurement()
        try:
            response = requests.post(API_URL, json=payload, timeout=3)
            print(f"[sim] -> {response.status_code} {response.text.strip()} payload={payload}")
        except Exception as exc:
            print(f"[sim] error: {exc}")
        time.sleep(INTERVAL_SEC)


if __name__ == "__main__":
    main()
