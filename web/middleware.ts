import { NextRequest, NextResponse } from 'next/server'

const LOCALE_SEGMENTS = new Set([
  'en',
  'fr',
  'es',
  'pt',
  'it',
  'de',
  'nl',
  'ru',
  'tr',
  'sw',
  'yo',
  'ig',
  'ha',
  'hi',
  'ja',
  'zh',
  'ar',
])

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl
  const [first, ...rest] = pathname.split('/').filter(Boolean)
  if (!first || !LOCALE_SEGMENTS.has(first)) {
    return NextResponse.next()
  }

  const url = req.nextUrl.clone()
  url.pathname = `/${rest.join('/')}` || '/'
  const res = NextResponse.rewrite(url)
  res.cookies.set('locale', first === 'zh' ? 'zh-Hans' : first, {
    path: '/',
    sameSite: 'lax',
    maxAge: 31536000,
  })
  return res
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|favicon.ico).*)'],
}
