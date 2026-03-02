export default function DegradedState({ message = 'Service degraded; showing best available data. Start backend stack (gateway/query/alerts + DB) to populate live views.' }: { message?: string }) {
  return <div className="empty-state">⚠️ {message}</div>
}
