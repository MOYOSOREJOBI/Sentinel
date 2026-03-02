import test from 'node:test'
import assert from 'node:assert/strict'
import { allLocales, dictionaries, t } from '../../lib/i18n'

test('i18n dictionary provides translated navigation labels', () => {
  assert.equal(t('fr', 'queue', 'Queue'), 'File priorisée')
  assert.equal(t('es', 'governance', 'Governance'), 'Gobernanza')
  assert.equal(t('pt', 'about', 'About'), 'Sobre')
  assert.equal(t('en', 'commandCenter', 'Command Center'), 'Command Center')
})

test('every locale contains every English key', () => {
  const englishKeys = Object.keys(dictionaries.en)
  assert.ok(englishKeys.length >= 60)
  for (const locale of allLocales()) {
    for (const key of englishKeys) {
      assert.equal(typeof dictionaries[locale][key], 'string', `${locale} missing ${key}`)
      assert.notEqual(dictionaries[locale][key], '', `${locale} empty ${key}`)
    }
  }
})
