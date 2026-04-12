#!/usr/bin/env python3
"""
第一阶段探测：拉取 Netdata GET /api/v1/charts，列出 Chart 与每个 chart 下 dimensions 的动态结构。

用法:
  ./scripts/netdata_probe_charts.py
  ./scripts/netdata_probe_charts.py --base http://192.168.10.206:20489
  ./scripts/netdata_probe_charts.py --json-out /tmp/netdata_charts.json   # 保存全文便于 jq
  ./scripts/netdata_probe_charts.py --sample-data system.cpu             # 再打一枪 data 看点数

依赖: 仅标准库 (urllib)。
"""
from __future__ import annotations

import argparse
import json
import sys
from typing import Any
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


def fetch_json(url: str, timeout: float = 15.0) -> Any:
    req = Request(url, headers={"Accept": "application/json"})
    with urlopen(req, timeout=timeout) as resp:
        return json.load(resp)


def classify_dimension_value(dim_id: str, val: Any) -> dict[str, Any]:
    """描述单个 dimension 条目的形态，便于与 Go 端 map[string]interface{} 策略对齐。"""
    row: dict[str, Any] = {"dimension_id": dim_id}
    if val is None:
        row["kind"] = "null"
        return row
    if isinstance(val, bool):
        row["kind"] = "bool"
        row["value"] = val
        return row
    if isinstance(val, (int, float)):
        row["kind"] = "number"
        row["value"] = float(val)
        return row
    if isinstance(val, str):
        row["kind"] = "string"
        row["sample"] = val[:120]
        return row
    if isinstance(val, list):
        row["kind"] = "array"
        row["len"] = len(val)
        row["elem_types"] = list({type(x).__name__ for x in val[:20]})
        return row
    if isinstance(val, dict):
        row["kind"] = "object"
        row["child_keys"] = sorted(val.keys())
        # 常见：只有 name；也可能有 multiplier、divisor、algorithm 等
        preview: dict[str, Any] = {}
        for k in row["child_keys"][:12]:
            v = val[k]
            if isinstance(v, (int, float)):
                preview[k] = ("number", float(v))
            elif isinstance(v, str):
                preview[k] = ("string", v[:60])
            elif isinstance(v, bool):
                preview[k] = ("bool", v)
            elif isinstance(v, dict):
                preview[k] = ("object", sorted(v.keys())[:8])
            else:
                preview[k] = (type(v).__name__, str(v)[:40])
        row["fields_preview"] = preview
        return row
    row["kind"] = type(val).__name__
    row["repr"] = repr(val)[:120]
    return row


def walk_charts(payload: Any) -> list[tuple[str, dict[str, Any]]]:
    """
    Netdata 常见两种顶层：整包就是 charts map，或 {"charts": {...}}。
    返回 [(chart_id, chart_obj), ...]
    """
    if isinstance(payload, dict) and "charts" in payload and isinstance(payload["charts"], dict):
        charts = payload["charts"]
    elif isinstance(payload, dict):
        # 若整 JSON 即 id -> chart
        charts = payload
        # 去掉明显非 chart 的顶层键
        skip = {"hostname", "version", "release_channel", "os", "timezone", "abbrev_timezone"}
        charts = {k: v for k, v in charts.items() if k not in skip and isinstance(v, dict)}
    else:
        return []

    out: list[tuple[str, dict[str, Any]]] = []
    for cid, cobj in charts.items():
        if not isinstance(cobj, dict):
            continue
        out.append((str(cid), cobj))
    return out


def main() -> int:
    ap = argparse.ArgumentParser(description="Probe Netdata /api/v1/charts")
    ap.add_argument(
        "--base",
        default="http://192.168.10.206:20489",
        help="Netdata 根 URL（无尾斜杠）",
    )
    ap.add_argument("--json-out", help="将原始 JSON 写入文件")
    ap.add_argument(
        "--max-charts",
        type=int,
        default=30,
        help="打印维度结构时最多抽样多少个 chart（0=不限制，慎用）",
    )
    ap.add_argument(
        "--sample-data",
        metavar="CHART_ID",
        help="额外请求 api/v1/data?chart=...&after=-1&points=1 展示数值列",
    )
    args = ap.parse_args()
    base = args.base.rstrip("/")
    charts_url = f"{base}/api/v1/charts"

    try:
        data = fetch_json(charts_url)
    except HTTPError as e:
        print(f"HTTP {e.code}: {e.reason}", file=sys.stderr)
        return 1
    except URLError as e:
        print(f"请求失败: {e}", file=sys.stderr)
        return 1

    if args.json_out:
        with open(args.json_out, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=2)
        print(f"已写入 {args.json_out}")

    pairs = walk_charts(data)
    print(f"URL: {charts_url}")
    print(f"解析到 chart 数量: {len(pairs)}")
    if not pairs:
        print("未识别 charts 结构，请检查 --json-out 保存的文件或 Netdata 版本。")
        return 2

    # 维度键形态统计
    kind_counts: dict[str, int] = {}
    charts_with_dims = 0
    total_dims = 0

    limit = len(pairs) if args.max_charts <= 0 else min(len(pairs), args.max_charts)
    print(f"\n--- 抽样前 {limit} 个 chart 的 dimensions 结构 ---\n")

    for cid, cobj in pairs[:limit]:
        dims = cobj.get("dimensions")
        if not isinstance(dims, dict):
            print(f"[{cid}] dimensions: {type(dims).__name__} (非 object，跳过)")
            continue
        charts_with_dims += 1
        title = cobj.get("title") or cobj.get("name") or ""
        print(f"### {cid}")
        if title:
            print(f"    title: {title}")
        print(f"    dimension 数量: {len(dims)}")
        for dk, dv in list(dims.items())[:12]:
            info = classify_dimension_value(dk, dv)
            k = info["kind"]
            kind_counts[k] = kind_counts.get(k, 0) + 1
            total_dims += 1
            extra = {x: info[x] for x in info if x not in ("dimension_id", "kind")}
            print(f"      - {dk!r}: kind={k} {extra}")
        if len(dims) > 12:
            print(f"      ... 另有 {len(dims) - 12} 个维度未展开")
        print()

    print("--- dimensions 值形态汇总（抽样行） ---")
    for k, v in sorted(kind_counts.items(), key=lambda x: -x[1]):
        print(f"  {k}: {v}")

    if args.sample_data:
        chart = args.sample_data
        data_url = f"{base}/api/v1/data?chart={chart}&after=-1&points=1&group=average"
        print(f"\n--- 样本 data: GET {data_url} ---")
        try:
            dresp = fetch_json(data_url)
        except (HTTPError, URLError) as e:
            print(f"失败: {e}", file=sys.stderr)
            return 0
        print(json.dumps(dresp, ensure_ascii=False, indent=2)[:4000])
        if len(json.dumps(dresp)) > 4000:
            print("\n... (截断，完整响应请自行 curl | jq)")

    print(
        "\n提示: 用 jq 浏览保存的文件示例:\n"
        f"  jq 'keys | length' {args.json_out or 'charts.json'}\n"
        "  jq '.charts | keys[:20]' charts.json\n"
        "  jq '.charts[\"system.cpu\"].dimensions' charts.json\n"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
