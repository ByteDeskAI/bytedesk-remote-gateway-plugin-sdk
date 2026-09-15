import { createComponentController, type ComponentHandle } from '../index.js'
const controller = createComponentController({
 identity: {id:'tasks:one',family:'tasks',ownerId:'projects',generation:'1'},
 snapshot:{available:true,running:false,href:'',starting:false,error:''},
 methods:{start:async()=>{}},
})
const handle: ComponentHandle<'tasks'> = controller.handle
handle.start?.()
handle.subscribe(s=>s.running, value=>{const running:boolean=value;void running})
// @ts-expect-error snapshots are readonly
handle.getSnapshot().running = true
// @ts-expect-error terminal input is not a component method
handle.write('whoami')
// @ts-expect-error foreign family methods are not present
handle.resize('one',1,1)
