#!/usr/bin/env python3
"""Generate golden JSON files for GoMDF tests using asammdf.

Run manually (requires: pip install asammdf numpy); outputs are committed
so CI never needs Python:

    python3 testdata/scripts/gen_golden.py samples/*.mf4
"""
import json
import math
import sys
from pathlib import Path

import numpy as np
from asammdf import MDF

OUT_DIR = Path(__file__).resolve().parent.parent / "golden"
HEAD = 10  # first/last N values recorded per channel


def encode(v):
    if isinstance(v, (bytes, np.bytes_)):
        return v.decode("utf-8", "replace")
    if isinstance(v, (np.floating, float)):
        f = float(v)
        if math.isnan(f):
            return None
        if math.isinf(f):
            return "+Inf" if f > 0 else "-Inf"
        return f
    if isinstance(v, (np.integer, int)):
        return int(v)
    return str(v)


def channel_entry(mdf, group_index, ch):
    sig = mdf.get(ch.name, group=group_index, raw=False)
    samples = sig.samples
    entry = {
        "group": group_index,
        "dtype": str(samples.dtype),
        "len": int(len(samples)),
        "first": [encode(v) for v in samples[:HEAD]],
        "last": [encode(v) for v in samples[-HEAD:]],
    }
    if samples.dtype.kind in "iuf":
        total = float(np.nansum(samples.astype("f8")))
        if math.isfinite(total):
            entry["sum"] = total
    return entry


def main(paths):
    OUT_DIR.mkdir(parents=True, exist_ok=True)
    for path in paths:
        path = Path(path)
        mdf = MDF(path)
        channels = {}
        for gi, group in enumerate(mdf.groups):
            for ch in group.channels:
                key = f"{gi}:{ch.name}"
                try:
                    channels[key] = channel_entry(mdf, gi, ch)
                except Exception as exc:  # record why a channel is skipped
                    channels[key] = {"group": gi, "error": str(exc)}
        out = OUT_DIR / (path.stem + ".json")
        out.write_text(json.dumps({"file": path.name, "channels": channels},
                                  indent=1, sort_keys=True))
        print(f"wrote {out}")


if __name__ == "__main__":
    main(sys.argv[1:] or sorted(Path("samples").glob("*.mf4")))
