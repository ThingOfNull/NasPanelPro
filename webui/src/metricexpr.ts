import { Parser } from 'expr-eval';

const parser = new Parser();

/** 按行 trim，丢弃空行（与 Go NonEmptyExprLines 一致） */
export function nonEmptyExprLines(s: string): string[] {
  if (!s.trim()) {
    return [];
  }
  const out: string[] = [];
  for (const line of s.split('\n')) {
    const t = line.trim();
    if (t) {
      out.push(t);
    }
  }
  return out;
}

function variableUnionFromLines(lines: string[]): string[] {
  const seen = new Set<string>();
  for (const line of lines) {
    try {
      const expr = parser.parse(line.trim());
      const vars = expr.variables() as string[];
      for (const v of vars) {
        seen.add(v);
      }
    } catch {
      // 忽略无法解析的行
    }
  }
  return [...seen].sort();
}

/** 复合模式：从 latest 中取出所有表达式行涉及变量的并集 */
export function buildLatestEnvForComposite(
  latest: Record<string, number>,
  lines: string[],
): Record<string, number> {
  const keys = variableUnionFromLines(lines);
  const env: Record<string, number> = {};
  for (const k of keys) {
    if (latest[k] !== undefined) {
      env[k] = latest[k];
    }
  }
  return env;
}

/** 复合模式：折线 series 子集 */
export function buildSeriesForComposite(
  series: Record<string, number[]>,
  lines: string[],
): Record<string, number[]> {
  const keys = variableUnionFromLines(lines);
  const out: Record<string, number[]> = {};
  for (const k of keys) {
    if (series[k]) {
      out[k] = series[k];
    }
  }
  return out;
}

function minSeriesLen(series: Record<string, number[]>): number {
  let min = -1;
  for (const pts of Object.values(series)) {
    if (pts.length === 0) {
      return 0;
    }
    if (min < 0 || pts.length < min) {
      min = pts.length;
    }
  }
  return min < 0 ? 0 : min;
}

/** 与 raypanel sanitizeLineSeriesForChart 一致的折线 NaN 处理 */
export function sanitizeLineSeriesForChart(pts: number[]): number[] {
  const n = pts.length;
  if (n === 0) {
    return pts;
  }
  const out = [...pts];
  for (let i = 1; i < n; i++) {
    if (!Number.isFinite(out[i])) {
      if (Number.isFinite(out[i - 1])) {
        out[i] = out[i - 1];
      }
    }
  }
  for (let i = n - 2; i >= 0; i--) {
    if (!Number.isFinite(out[i])) {
      if (Number.isFinite(out[i + 1])) {
        out[i] = out[i + 1];
      }
    }
  }
  for (let i = 0; i < n; i++) {
    if (!Number.isFinite(out[i])) {
      out[i] = 0;
    }
  }
  return out;
}

export function evalScalar(exprStr: string, env: Record<string, number>): number {
  const trimmed = exprStr.trim();
  if (!trimmed) {
    throw new Error('empty expression');
  }
  const expr = parser.parse(trimmed);
  const v = expr.evaluate(env) as number;
  return typeof v === 'number' && Number.isFinite(v) ? v : NaN;
}

export function evalSeries(
  exprStr: string,
  series: Record<string, number[]>,
): number[] {
  const trimmed = exprStr.trim();
  if (!trimmed) {
    throw new Error('empty expression');
  }
  const n = minSeriesLen(series);
  if (n === 0) {
    return [];
  }
  const expr = parser.parse(trimmed);
  const out: number[] = [];
  for (let i = 0; i < n; i++) {
    const env: Record<string, number> = {};
    for (const k of Object.keys(series)) {
      const pts = series[k];
      if (i < pts.length) {
        env[k] = pts[i];
      }
    }
    try {
      const v = expr.evaluate(env) as number;
      out.push(typeof v === 'number' && Number.isFinite(v) ? v : NaN);
    } catch {
      out.push(NaN);
    }
  }
  return sanitizeLineSeriesForChart(out);
}
