import assert from 'node:assert/strict'
import { test } from 'node:test'
import { isDesktopApplication, isDesktopApplicationsScanV2Request, isDesktopApplicationsScanV2Result } from './index.js'

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
    executablePath: '/opt/claude/claude',
    installedAt: '2026-09-12T18:30:00Z',
    installedAtEstimated: true
  }), true)

  assert.equal(isDesktopApplication({ ...application, launcherPath: 42 }), false)
  assert.equal(isDesktopApplication({ ...application, executablePath: 42 }), false)
  assert.equal(isDesktopApplication({ ...application, executablePath: null }), false)
  assert.equal(isDesktopApplication({ ...application, installedAt: false }), false)
  assert.equal(isDesktopApplication({ ...application, installedAtEstimated: 'yes' }), false)
})

test('desktop application scan v2 browser contract carries async pages', () => {
  assert.equal(isDesktopApplicationsScanV2Request({}), true)
  assert.equal(isDesktopApplicationsScanV2Request({ scanId: 'scan-1', cursor: 'page-2', limit: 100 }), true)
  assert.equal(isDesktopApplicationsScanV2Request({ limit: '100' }), false)

  const complete = {
    scanId: 'scan-1',
    state: 'complete',
    revision: 'revision-1',
    scannedAt: '2026-09-13T16:00:00Z',
    total: 1,
    applications: [application]
  }
  assert.equal(isDesktopApplicationsScanV2Result(complete), true)
  // Generated browser guards check structure only. The Go Validate method
  // enforces the stronger non-null application-page invariant at the host.
  const { applications: _, ...missingApplications } = complete
  assert.equal(isDesktopApplicationsScanV2Result(missingApplications), false)
  assert.equal(isDesktopApplicationsScanV2Result({ ...complete, total: '1' }), false)
})
