import type { AppRouterInstance } from 'next/dist/shared/lib/app-router-context.shared-runtime'
import { api } from './api'

export async function requireAuth(router: AppRouterInstance) {
  try {
    return await api.me()
  } catch (err: any) {
    if (err?.status === 401 || String(err?.message || '').includes('401')) {
      router.replace('/login')
      return null
    }
    throw err
  }
}
