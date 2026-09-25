import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

test('AI browser exports retain canonical concrete schemas', async () => {
 const sidecar=JSON.parse(fs.readFileSync(new URL('./v2/schemas.json',import.meta.url),'utf8'))
 for (const pkg of ['aidecision','payloads','provideraccess','codingsessions','hostsettings']) {
  const {descriptors}=await import(`@bytedesk/gateway-plugin-ui/v2/${pkg}/descriptors`)
  const validators=await import(`@bytedesk/gateway-plugin-ui/v2/${pkg}/contracts`)
  assert.ok(Object.keys(validators).some(key=>key.startsWith('is')))
  for(const descriptor of Object.values(descriptors)){
   assert.match(descriptor.schema,/^[a-f0-9]{64}$/)
   assert.equal(sidecar.operations[descriptor.name][String(descriptor.rev)].hash,descriptor.schema)
  }
 }
})
