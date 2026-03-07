export default function DegradedState({
  message = 'Service degraded; showing best available data. Start backend stack (gateway/query/alerts + DB) to populate live views.',
  onRetry,
}: {
  message?: string
  onRetry?: () => void
}) {
  return <div className="empty-state" style={{ display: 'flex', alignItems: 'center', gap: 10, justifyContent: 'space-between' }}>
    <span>⚠️ {message}</span>
    {onRetry ? <button type="button" className="ghost-btn small" onClick={onRetry}>Retry</button> : null}
  </div>
}
