import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isDesktopApplication } from './index.js'

const application = {
  id: 'claude-desktop',
  name: 'Claude Desktop',
  kind: 'desktop',
  status: 'ready'
}

test('desktop application validator exposes optional catalog metadata', () => {
  assert.equal(isDesktopApplication(application), true)
  assert.equal(isDesktopApplication({
    ...application,
    iconUrl: '/api/plugins/applications/icons/claude-desktop',
    launcherPath: '/usr/share/applications/claude.desktop',
    installedAt: '2026-09-12T18:30:00Z',
    installedAtEstimated: true
  }), true)

  assert.equal(isDesktopApplication({ ...application, launcherPath: 42 }), false)
  assert.equal(isDesktopApplication({ ...application, installedAt: false }), false)
  assert.equal(isDesktopApplication({ ...application, installedAtEstimated: 'yes' }), false)
})
