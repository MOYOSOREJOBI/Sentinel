export default function EmptyState({ message = 'No data for current filters.' }: { message?: string }) {
  return <div className="empty-state">{message}</div>
}
