'use client'
import { useEffect, useState } from 'react'
import AppShell, { useGlobalFilters } from '../../components/AppShell'
import { api } from '../../lib/api'

export default function ExecutivePage() {
  const [summary, setSummary] = useState<any>({})
  const { filters, apply } = useGlobalFilters()
  useEffect(() => { api.executiveSummary(filters).then(setSummary) }, [JSON.stringify(filters)])

  function exportBrief() {
    const ts = new Date().toISOString()
    const topRisks = (summary.topRisks || []).map((r: any) =>
      `- ${r.symbol} | ${r.severityBand || '-'} | priority ${Number(r.priorityScore || 0).toFixed(2)} | composite ${Number(r.compositeRisk || 0).toFixed(2)}`
    ).join('\n')
    const memo = summary.riskMemo || 'No material change.'
    const text = `# Sentinel Executive Brief\n_Generated: ${ts}_\n\n## Risk Memo\n${memo}\n\n## Top Risks\n${topRisks || 'None in window.'}\n\n## Country Concentration\n${(summary.countryConcentration || []).length} countries with active incidents.\n\n## Industry Concentration\n${(summary.industryConcentration || []).length} industries with active incidents.`
    navigator.clipboard.writeText(text).catch(() => {})
  }

  return <AppShell title="Executive Summary" subtitle="Plain-English current risk posture" titleKey="titleExecutive" subtitleKey="subtitleExecutive" filters={filters} setFilters={apply}>
    <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '0.75rem' }}>
      <button
        data-testid="export-brief"
        onClick={exportBrief}
        style={{ padding: '0.35rem 1rem', borderRadius: '0.5rem', border: '1px solid var(--border, #334)', background: 'transparent', cursor: 'pointer' }}
      >
        Export Brief
      </button>
    </div>
    <div className="grid-3">
      <div className="card"><h3>Top 5 risks</h3><p>{(summary.topRisks || []).length}</p></div>
      <div className="card"><h3>Country concentration</h3><p>{(summary.countryConcentration || []).length}</p></div>
      <div className="card"><h3>Industry concentration</h3><p>{(summary.industryConcentration || []).length}</p></div>
    </div>
    <section className="card">
      <h3>Top risk table</h3>
      <table className="table">
        <thead><tr><th>ID</th><th>Symbol</th><th>Priority</th><th>Composite</th><th>Severity</th></tr></thead>
        <tbody>
          {(summary.topRisks || []).map((r: any) => (
            <tr key={r.id}><td>{r.id}</td><td>{r.symbol}</td><td>{Number(r.priorityScore || 0).toFixed(2)}</td><td>{Number(r.compositeRisk || 0).toFixed(2)}</td><td>{r.severityBand || '-'}</td></tr>
          ))}
        </tbody>
      </table>
    </section>
    <section className="card"><h3>Risk memo</h3><p>{summary.riskMemo || 'No material change.'}</p></section>
  </AppShell>
}
