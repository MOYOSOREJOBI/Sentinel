import { proxySSE } from '../proxy'

const API_ALERTS = process.env.UPSTREAM_ALERTS || process.env.API_ALERTS || 'http://localhost:8083'

export const dynamic = 'force-dynamic'

export async function GET(request: Request) {
  return proxySSE(request, `${API_ALERTS}/sse/alerts`, 'alert')
}
