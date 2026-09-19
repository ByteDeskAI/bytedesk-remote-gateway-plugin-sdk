import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

import * as descriptors from '@bytedesk/gateway-plugin-ui/v2/descriptors'
import * as validators from '@bytedesk/gateway-plugin-ui/v2/contracts'
import { descriptors as webAppDescriptors } from '@bytedesk/gateway-plugin-ui/v2/webapps/descriptors'
import * as webAppValidators from '@bytedesk/gateway-plugin-ui/v2/webapps/contracts'

test('v2 package exports generated contracts, descriptors and schemas', () => {
  const pkg = JSON.parse(fs.readFileSync(new URL('../package.json', import.meta.url), 'utf8'))
  const version = fs.readFileSync(new URL('../v2/VERSION', import.meta.url), 'utf8').trim()
  assert.equal(pkg.version, version)
  assert.ok(Object.keys(validators).some(name => name.startsWith('is')))
  assert.ok(Object.keys(descriptors).length > 0)
  const schemas = JSON.parse(fs.readFileSync(new URL('./v2/schemas.json', import.meta.url), 'utf8'))
  assert.ok(schemas && typeof schemas === 'object')
  assert.equal(webAppDescriptors.List.name, 'cmd.web-apps.v1.list')
  assert.equal(webAppDescriptors.OpenPreviewExternal.name, 'cmd.web-apps.v1.preview.open-external')
  assert.equal(typeof webAppValidators.isConversationSendRequest, 'function')
})
