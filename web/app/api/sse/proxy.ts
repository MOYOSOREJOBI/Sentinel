const SSE_HEADERS = {
  'Content-Type': 'text/event-stream',
  'Cache-Control': 'no-cache, no-transform',
  Connection: 'keep-alive',
}

function degradedPayload(eventName: string, message: string) {
  const enc = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      controller.enqueue(enc.encode('retry: 5000\n\n'))
      controller.enqueue(enc.encode(`event: ${eventName}\ndata: ${JSON.stringify({ type: 'degraded', message })}\n\n`))
      controller.close()
    },
  })
}

function humanizeStreamError(err: unknown) {
  const raw = typeof (err as any)?.message === 'string' ? String((err as any).message) : 'upstream unavailable'
  if (raw.startsWith('upstream ')) {
    return 'Live updates paused. Reconnecting to the stream.'
  }
  if (raw.toLowerCase().includes('terminated') || raw.toLowerCase().includes('socket')) {
    return 'Live updates paused. Reconnecting to the stream.'
  }
  return 'Live updates unavailable. Retrying shortly.'
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
    const reader = upstream.body.getReader()
    const enc = new TextEncoder()
    const stream = new ReadableStream({
      async pull(controller) {
        try {
          const { done, value } = await reader.read()
          if (done) {
            controller.close()
            return
          }
          if (value) {
            controller.enqueue(value)
          }
        } catch (err) {
          controller.enqueue(enc.encode(`event: ${eventName}\ndata: ${JSON.stringify({ type: 'degraded', message: humanizeStreamError(err) })}\n\n`))
          controller.close()
        }
      },
      cancel() {
        void reader.cancel().catch(() => null)
      },
    })
    request.signal.addEventListener('abort', () => {
      void reader.cancel().catch(() => null)
    })
    return new Response(stream, { status: 200, headers: SSE_HEADERS })
  } catch (err: any) {
    return new Response(degradedPayload(eventName, humanizeStreamError(err)), { status: 200, headers: SSE_HEADERS })
  }
}
