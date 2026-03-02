import { proxySSE } from '../proxy'

const API_QUERY = process.env.UPSTREAM_QUERY || process.env.API_QUERY || 'http://localhost:8085'

export const dynamic = 'force-dynamic'

export async function GET(request: Request) {
  return proxySSE(request, `${API_QUERY}/stream/command-center`, 'command_center_patch')
}
