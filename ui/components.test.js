import {test} from 'node:test'
import assert from 'node:assert/strict'
import {createComponentController,connectComponent,componentChangedEvent} from './components.js'
const identity = {id:'tasks:one',family:'tasks',ownerId:'projects',generation:'1',projectId:'one'}
const snapshot = {available:true,running:false,href:'',starting:false,error:''}
test('immutable snapshots, selective notification and supported methods', async()=>{
 let starts=0,changes=0,running=0
 const source={...snapshot}
 const c=createComponentController({identity,snapshot:source,methods:{start:async()=>{starts++}}})
 source.running=true
 assert.equal(c.handle.getSnapshot().running,false)
 assert.throws(()=>{c.handle.getSnapshot().running=true},TypeError)
 c.handle.subscribe(()=>changes++)
 c.handle.subscribe(s=>s.running,()=>running++)
 c.update({...snapshot,starting:true});c.update({...snapshot,starting:true})
 assert.equal(changes,1);assert.equal(running,0)
 c.update({...snapshot,running:true});assert.equal(running,1)
 assert.deepEqual(c.handle.capabilities,['start']);assert.equal(c.handle.refresh,undefined)
 await c.handle.start();assert.equal(starts,1)
 c.dispose();assert.throws(()=>c.handle.getSnapshot(),/disposed/);await assert.rejects(c.handle.start(),/disposed/)
})
test('rejects replacement and terminal IO, lifetime abort invalidates handles',()=>{
 assert.throws(()=>createComponentController({identity,snapshot,methods:{write:()=>{}}}),/Unsupported/)
 const lifetime=new AbortController()
 const c=createComponentController({identity,snapshot,signal:lifetime.signal});lifetime.abort()
 assert.throws(()=>c.update(snapshot),/disposed/)
})
test('client uses exact assignment, ignores other generations, cleans subscription on withdrawal',async()=>{
 const lifetime=new AbortController();let event,unsubscribed=0;const calls=[]
 const host={signal:lifetime.signal,subscribe(type,callback){assert.equal(type,componentChangedEvent);event=callback;return()=>unsubscribed++},async request(command,payload){calls.push({command,payload});if(command.includes('snapshot'))return{identity,revision:1,snapshot,capabilities:['start']}}}
 const assignment={identity,lease:'opaque-host-assignment'}
 const handle=await connectComponent(host,assignment)
 await handle.start();assert.deepEqual(calls[1].payload,{assignment,method:'start',args:[]})
 event({identity:{...identity,generation:'2'},lease:assignment.lease,snapshot:{...snapshot,running:true}})
 assert.equal(handle.getSnapshot().running,false)
 event({identity,revision:2,lease:assignment.lease,snapshot:{...snapshot,running:true}});assert.equal(handle.getSnapshot().running,true)
 event({identity,lease:assignment.lease,withdrawn:true});assert.equal(unsubscribed,1);await assert.rejects(handle.start(),/disposed/)
})
test('rejects missing assignment, identity substitution, and unknown capabilities',async()=>{
 const signal=new AbortController().signal;let cleaned=0
 const host={signal,subscribe(){return()=>cleaned++},async request(){return{identity:{...identity,id:'other'},snapshot,capabilities:[]}}}
 await assert.rejects(connectComponent(host,{identity}),/assignment/)
 await assert.rejects(connectComponent(host,{identity,lease:'x'}),/mismatch/);assert.equal(cleaned,1)
 host.request=async()=>({identity,revision:1,snapshot,capabilities:['write']})
 await assert.rejects(connectComponent(host,{identity,lease:'x'}),/Unsupported/);assert.equal(cleaned,2)
})
test('host abort during initial read cannot leak a subscription',async()=>{
 const lifetime=new AbortController();let finish,cleaned=0
 const host={signal:lifetime.signal,subscribe(){return()=>cleaned++},request(){return new Promise(resolve=>{finish=resolve})}}
 const pending=connectComponent(host,{identity,lease:'x'});lifetime.abort();finish({identity,revision:1,snapshot,capabilities:[]})
 await assert.rejects(pending,/disposed/);assert.equal(cleaned,1)
})
test('malformed remote snapshots withdraw rather than enter component state',async()=>{
 let event,cleaned=0
 const host={signal:new AbortController().signal,subscribe(_,fn){event=fn;return()=>cleaned++},async request(){return{identity,revision:1,snapshot,capabilities:[]}}}
 const handle=await connectComponent(host,{identity,lease:'x'})
 event({identity,revision:2,lease:'x',snapshot:{available:'yes'}})
 assert.equal(cleaned,1);assert.throws(()=>handle.getSnapshot(),/disposed/)
})
test('initial response and queued update are structurally validated',async()=>{
 const host={signal:new AbortController().signal,subscribe(){return()=>{}},async request(){return{identity,revision:1,snapshot:{available:true},capabilities:[]}}}
 await assert.rejects(connectComponent(host,{identity,lease:'x'}),/snapshot/)
})
test('named methods reject invalid arguments before adapter execution',async()=>{
 let calls=0
 const c=createComponentController({identity:{...identity,family:'stage'},snapshot:{placements:[],saveStatus:'saved'},methods:{resize:async()=>{calls++},move:async()=>{calls++}}})
 await assert.rejects(c.handle.resize('one',3,1),/Invalid arguments/)
 await assert.rejects(c.handle.move('one',2,0),/Invalid arguments/)
 await assert.rejects(c.handle.move('one',0,-1),/Invalid arguments/)
 assert.equal(calls,0);await c.handle.resize('one',2,1);assert.equal(calls,1)
})
test('disposal invalidates a command that completes after its instance disappeared',async()=>{
 let complete
 const c=createComponentController({identity,snapshot,methods:{start:()=>new Promise(resolve=>{complete=resolve})}})
 const pending=c.handle.start();c.dispose();complete()
 await assert.rejects(pending,/disposed/)
})

test('initial snapshot revision rejects an older queued update',async()=>{
 let event,resolve
 const host={signal:new AbortController().signal,subscribe(_,fn){event=fn;return()=>{}},request(){return new Promise(done=>{resolve=done})}}
 const pending=connectComponent(host,{identity,lease:'x'})
 event({identity,lease:'x',revision:1,snapshot:{...snapshot,running:false}})
 resolve({identity,revision:2,snapshot:{...snapshot,running:true},capabilities:[]})
 const handle=await pending;assert.equal(handle.getSnapshot().running,true)
 event({identity,lease:'x',revision:2,snapshot:{...snapshot,running:false}})
 assert.equal(handle.getSnapshot().running,true);handle.dispose()
})
test('queued newer revisions apply in order after an older initial snapshot',async()=>{
 let event,resolve
 const host={signal:new AbortController().signal,subscribe(_,fn){event=fn;return()=>{}},request(){return new Promise(done=>{resolve=done})}}
 const pending=connectComponent(host,{identity,lease:'x'})
 event({identity,lease:'x',revision:3,snapshot:{...snapshot,running:true}})
 event({identity,lease:'x',revision:2,snapshot:{...snapshot,running:false}})
 resolve({identity,revision:1,snapshot,capabilities:[]})
 const handle=await pending;assert.equal(handle.getSnapshot().running,true);handle.dispose()
})
test('invalid live revision fails closed; queued withdrawal dominates snapshot',async()=>{
 for(const revision of [undefined,-1,1.5,Number.MAX_SAFE_INTEGER+1]) {
  const host={signal:new AbortController().signal,subscribe(){return()=>{}},async request(){return{identity,revision,snapshot,capabilities:[]}}}
  await assert.rejects(connectComponent(host,{identity,lease:'x'}),/revision/)
 }
 let cleaned=0
 const host={signal:new AbortController().signal,subscribe(_,fn){fn({identity,lease:'x',withdrawn:true,revision:0});return()=>cleaned++},async request(){return{identity,revision:1,snapshot,capabilities:[]}}}
 await assert.rejects(connectComponent(host,{identity,lease:'x'}),/withdrawn/);assert.equal(cleaned,1)
})
