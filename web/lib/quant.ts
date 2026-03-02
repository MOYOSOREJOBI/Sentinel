export function clamp01(v: number) {
  if (Number.isNaN(v)) return 0
  if (v < 0) return 0
  if (v > 1) return 1
  return v
}

function mean(values: number[]) {
  if (!values.length) return 0
  return values.reduce((a, b) => a + b, 0) / values.length
}

export function percentile(values: number[], q: number) {
  if (!values.length) return 0
  const sorted = [...values].sort((a, b) => a - b)
  const idx = Math.min(sorted.length - 1, Math.max(0, Math.floor(q * (sorted.length - 1))))
  return sorted[idx]
}

export function stddev(values: number[]) {
  if (values.length < 2) return 0
  const m = mean(values)
  const variance = mean(values.map((v) => (v - m) ** 2))
  return Math.sqrt(variance)
}

export function deriveQuantMetrics(rows: any[]) {
  const composite = rows.map((r) => Number(r.compositeRisk || r.composite_risk || 0))
  const priority = rows.map((r) => Number(r.priorityScore || r.priority_score || 0))
  const confidence = rows.map((r) => Number(r.confidence || 0))
  const escalation = rows.map((r) => Number(r.escalationProbability || r.escalation_probability || 0))

  const meanComposite = mean(composite)
  const volComposite = stddev(composite)
  const p95Composite = percentile(composite, 0.95)
  const meanPriority = mean(priority)
  const meanEscalation = mean(escalation)
  const confidenceWeightedRisk = mean(
    rows.map((r) => Number(r.compositeRisk || r.composite_risk || 0) * Number(r.confidence || 0)),
  )
  const expectedLossProxy = mean(
    rows.map((r) => Number(r.priorityScore || r.priority_score || 0) * Number(r.escalationProbability || r.escalation_probability || 0)),
  )
  const signalToNoise = meanComposite / Math.max(1e-6, volComposite)

  return {
    meanComposite,
    volComposite,
    p95Composite,
    meanPriority,
    meanEscalation,
    confidenceWeightedRisk,
    expectedLossProxy,
    signalToNoise,
    avgConfidence: mean(confidence),
  }
}
