package natsconn

import (
	"context"
	"io"
	"time"

	"github.com/ByteDeskAI/bytedesk-sdk-dependencies/v2/bus"
)

// The durable surfaces this transport does not implement yet: streams, KV,
// objects and schedules.
//
// They answer bus.Unsupported on every call rather than panicking on a nil
// interface, but the refusal is the SECOND line of defence. The first is
// Capabilities: TransportCapabilities reports these false, the negotiated set
// is intersected with it, and a manifest that NEEDS one fails closed at the
// handshake naming the capability. A plugin should never reach these methods.
//
// ponytail: the JetStream implementations land with the durable re-platforms
// (plan 3). Implement them here and flip the flags in TransportCapabilities;
// nothing else in the SDK changes.

func unsupportedFault(op, capability string) bus.Fault { return bus.Unsupported(op, capability) }

// noStreams is bus.Streams.
type noStreams struct{}

func (noStreams) Declare(context.Context, bus.StreamSpec) error {
	return unsupportedFault("streams.declare", "durable")
}
func (noStreams) Publish(context.Context, bus.Subject, []byte, ...bus.PublishOpt) (bus.Seq, error) {
	return 0, unsupportedFault("streams.publish", "durable")
}
func (noStreams) PublishBatch(context.Context, []bus.BatchMsg) ([]bus.Seq, error) {
	return nil, unsupportedFault("streams.publishBatch", "batch")
}
func (noStreams) Counter(context.Context, bus.Subject, int64) (int64, error) {
	return 0, unsupportedFault("streams.counter", "counters")
}
func (noStreams) Consume(context.Context, string, bus.ConsumerSpec, bus.StreamHandler) (bus.Consumer, error) {
	return nil, unsupportedFault("streams.consume", "durable")
}
func (noStreams) Fetch(context.Context, string, bus.ConsumerSpec, int) ([]*bus.StreamMsg, error) {
	return nil, unsupportedFault("streams.fetch", "durable")
}
func (noStreams) Purge(context.Context, string, bus.Pattern) error {
	return unsupportedFault("streams.purge", "durable")
}
func (noStreams) Info(context.Context, string) (bus.StreamInfo, error) {
	return bus.StreamInfo{}, unsupportedFault("streams.info", "durable")
}
func (noStreams) Delete(context.Context, string) error {
	return unsupportedFault("streams.delete", "durable")
}

// noKV is bus.KV.
type noKV struct{}

func (noKV) Declare(context.Context, bus.BucketSpec) error {
	return unsupportedFault("kv.declare", "kv")
}
func (noKV) Open(context.Context, string) (bus.Bucket, error) {
	return nil, unsupportedFault("kv.open", "kv")
}
func (noKV) Delete(context.Context, string) error   { return unsupportedFault("kv.delete", "kv") }
func (noKV) List(context.Context) ([]string, error) { return nil, unsupportedFault("kv.list", "kv") }

// noObjects is bus.Objects.
type noObjects struct{}

func (noObjects) Declare(context.Context, bus.BucketSpec) error {
	return unsupportedFault("objects.declare", "objects")
}
func (noObjects) Put(context.Context, bus.ObjectMeta, io.Reader) (bus.ObjectMeta, error) {
	return bus.ObjectMeta{}, unsupportedFault("objects.put", "objects")
}
func (noObjects) Get(context.Context, string, string) (io.ReadCloser, bus.ObjectMeta, error) {
	return nil, bus.ObjectMeta{}, unsupportedFault("objects.get", "objects")
}
func (noObjects) Info(context.Context, string, string) (bus.ObjectMeta, error) {
	return bus.ObjectMeta{}, unsupportedFault("objects.info", "objects")
}
func (noObjects) Delete(context.Context, string, string) error {
	return unsupportedFault("objects.delete", "objects")
}
func (noObjects) List(context.Context, string) ([]bus.ObjectMeta, error) {
	return nil, unsupportedFault("objects.list", "objects")
}
func (noObjects) Watch(context.Context, string) (bus.ObjectWatcher, error) {
	return nil, unsupportedFault("objects.watch", "objects")
}

// noScheduler is bus.Scheduler.
//
// Cancel is deliberately NOT a fault: Stop calls it, Stop must be safe to call
// twice, and a teardown path that reports "unsupported" for work that was never
// scheduled turns every clean shutdown into a logged error.
type noScheduler struct{}

func (noScheduler) At(context.Context, string, time.Time, bus.Subject, []byte, ...bus.PublishOpt) error {
	return unsupportedFault("schedule.at", "schedule")
}
func (noScheduler) Every(context.Context, string, time.Duration, bus.Subject, []byte, ...bus.PublishOpt) error {
	return unsupportedFault("schedule.every", "schedule")
}
func (noScheduler) Cron(context.Context, string, string, bus.Subject, []byte, ...bus.PublishOpt) error {
	return unsupportedFault("schedule.cron", "schedule")
}
func (noScheduler) Cancel(context.Context, string) error { return nil }
func (noScheduler) List(context.Context) ([]bus.Schedule, error) {
	return nil, unsupportedFault("schedule.list", "schedule")
}
