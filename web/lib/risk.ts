export type ScoreSeverity = 'critical' | 'high' | 'medium' | 'low' | string

export function summarizeScoreSeverity(scores: Array<{ severity: ScoreSeverity }>) {
  let critical = 0
  let high = 0
  for (const score of scores) {
    if (score.severity === 'critical') critical += 1
    if (score.severity === 'high') high += 1
  }
  return { critical, high }
}
