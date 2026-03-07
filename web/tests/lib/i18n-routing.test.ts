import test from 'node:test'
import assert from 'node:assert/strict'
import { localeFromRoute, localeLabel, localeToRoute, stripLocalePrefix, withLocalePath } from '../../lib/i18n'

test('route locale helpers translate zh alias and strip prefixes', () => {
  assert.equal(localeFromRoute('zh'), 'zh-Hans')
  assert.equal(localeToRoute('zh-Hans'), 'zh')
  assert.equal(stripLocalePrefix('/ar/command-center'), '/command-center')
  assert.equal(stripLocalePrefix('/queue'), '/queue')
})

test('withLocalePath prefixes logical routes', () => {
  assert.equal(withLocalePath('/command-center', 'fr'), '/fr/command-center')
  assert.equal(withLocalePath('/zh/queue', 'ar'), '/ar/queue')
  assert.equal(withLocalePath('/', 'zh-Hans'), '/zh')
})

test('locale labels stay deterministic across server and client environments', () => {
  assert.equal(localeLabel('yo'), 'Yorùbá')
  assert.equal(localeLabel('zh-Hans'), '简体中文')
})
