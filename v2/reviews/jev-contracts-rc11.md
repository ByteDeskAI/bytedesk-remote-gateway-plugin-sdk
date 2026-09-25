# Gateway AI SDK release review

Date: 2026-09-25. Gateway task: TM-475 / EP-028.

Independent review approved the rc11 release candidate. Review covered the five
pure common-contract re-export packages, typed helpers, generic public-boundary
inventory, canonical browser artifacts and sync command, workload negotiation
and transport, and both aggregate host-readiness features.

Verified workload safeguards: generation agreement, consistent feature/token
negotiation, refusal of injected transports without token binding, and token
stripping on Publish, Respond and requests outside the host command namespace.

Verification passed: root Go suite, full v2 race suite, focused independent
boundary/workload rerun, browser canonical artifact checks and all 26 npm tests.
An earlier transport dial timeout passed five isolated race reruns and the final
full race suite; it is not recorded as an accepted failing test.

The host must enforce private negotiation replies, ambiguous-header refusal,
current workload admission and host-only provider ingress before advertising
readiness. This SDK release alone does not prove host or Store delivery.
