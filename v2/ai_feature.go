package pluginsdk

// FeatureAIDecisionV1 is an aggregate host readiness promise. Advertise it only
// when all of these exist and are tested: the typed decision facade; automatic
// enabled-declaring-consumer admission; host-issued invocation scopes; bounded
// payload and credential/egress isolation; host-only provider ingress ACLs; and
// host-owned secret settings. A partial implementation must not advertise it.
// Providers require this feature AND FeatureHostWorkloadAuth so an older host fails
// activation rather than running with a reduced trust boundary.
const FeatureAIDecisionV1 = "ai.decision.v1"

// FeatureCodingSessionsV1 promises the full host-owned coding boundary:
// durable task lifecycle and ordered recovery; task-bound provider/model/config
// routing; enforceable permission modes; isolated committed-checkout worktrees;
// and Projects/dock attachment to the same session with explicit close ending it.
// Explicit originating Task Management links are host-authorized and immutable;
// authoritative completion ends the linked shared session. Unlinked tasks remain
// supported, and new tasks never silently inherit an earlier work-unit link.
// Hosts must not advertise it for a partial command or UI-only implementation.
const FeatureCodingSessionsV1 = "coding.sessions.v1"
