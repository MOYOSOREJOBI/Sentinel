const SSE_HEADERS = {
  'Content-Type': 'text/event-stream',
  'Cache-Control': 'no-cache, no-transform',
  Connection: 'keep-alive',
}

export async function proxySSE(request: Request, upstreamURL: string, eventName: string) {
  const cookie = request.headers.get('cookie') || ''
  try {
    const upstream = await fetch(upstreamURL, {
      cache: 'no-store',
      headers: {
        Accept: 'text/event-stream',
        Cookie: cookie,
      },
    })
    if (!upstream.ok || !upstream.body) {
      throw new Error(`upstream ${upstream.status}`)
    }
    return new Response(upstream.body, { status: 200, headers: SSE_HEADERS })
  } catch (err: any) {
    const message = typeof err?.message === 'string' ? err.message : 'upstream unavailable'
    const enc = new TextEncoder()
    const stream = new ReadableStream({
      start(controller) {
        controller.enqueue(enc.encode('retry: 15000\n\n'))
        controller.enqueue(enc.encode(`event: ${eventName}\ndata: ${JSON.stringify({ type: 'degraded', message })}\n\n`))
        const timer = setInterval(() => {
          controller.enqueue(enc.encode(`event: ${eventName}\ndata: ${JSON.stringify({ type: 'degraded-heartbeat', message: 'waiting for upstream' })}\n\n`))
        }, 15000)
        request.signal.addEventListener('abort', () => {
          clearInterval(timer)
          controller.close()
        })
      },
    })
    return new Response(stream, { status: 200, headers: SSE_HEADERS })
  }
}
