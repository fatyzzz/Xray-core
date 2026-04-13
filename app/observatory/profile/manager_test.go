package profile

import (
	"context"
	"testing"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/features/extension"
	"google.golang.org/protobuf/proto"
)

type fakeObservatory struct {
	result *observatory.ObservationResult
}

func (f *fakeObservatory) Type() interface{} { return extension.ObservatoryType() }
func (f *fakeObservatory) Start() error      { return nil }
func (f *fakeObservatory) Close() error      { return nil }
func (f *fakeObservatory) GetObservation(context.Context) (proto.Message, error) {
	return f.result, nil
}

func TestManagerGetObservationUsesFirstObservatoryAsDefault(t *testing.T) {
	m := &Manager{
		defaultTag: "youtube",
		observatories: map[string]extension.Observatory{
			"youtube": &fakeObservatory{
				result: &observatory.ObservationResult{
					Status: []*observatory.OutboundStatus{{OutboundTag: "yt-a", Alive: true, Delay: 42}},
				},
			},
			"global": &fakeObservatory{
				result: &observatory.ObservationResult{
					Status: []*observatory.OutboundStatus{{OutboundTag: "proxy-a", Alive: true, Delay: 420}},
				},
			},
		},
	}

	got, err := m.GetObservation(context.Background())
	if err != nil {
		t.Fatalf("GetObservation() failed: %v", err)
	}

	result := got.(*observatory.ObservationResult)
	if len(result.Status) != 1 || result.Status[0].OutboundTag != "yt-a" {
		t.Fatalf("unexpected default observatory result: %+v", result.Status)
	}
}

func TestManagerGetObservationByTagIsScoped(t *testing.T) {
	m := &Manager{
		defaultTag: "youtube",
		observatories: map[string]extension.Observatory{
			"youtube": &fakeObservatory{
				result: &observatory.ObservationResult{
					Status: []*observatory.OutboundStatus{{OutboundTag: "yt-a", Alive: true, Delay: 42}},
				},
			},
			"global": &fakeObservatory{
				result: &observatory.ObservationResult{
					Status: []*observatory.OutboundStatus{{OutboundTag: "proxy-a", Alive: true, Delay: 420}},
				},
			},
		},
	}

	got, err := m.GetObservationByTag(context.Background(), "global")
	if err != nil {
		t.Fatalf("GetObservationByTag() failed: %v", err)
	}

	result := got.(*observatory.ObservationResult)
	if len(result.Status) != 1 || result.Status[0].OutboundTag != "proxy-a" {
		t.Fatalf("unexpected tagged observatory result: %+v", result.Status)
	}
}

func TestManagerGetObservationByTagForUnknownObservatoryFails(t *testing.T) {
	m := &Manager{
		defaultTag:    "youtube",
		observatories: map[string]extension.Observatory{},
	}

	if _, err := m.GetObservationByTag(context.Background(), "missing"); err == nil {
		t.Fatal("expected unknown observatory tag to fail")
	}
}

func TestNewNilConfigFails(t *testing.T) {
	if _, err := New(context.Background(), nil); err == nil {
		t.Fatal("expected nil config to fail")
	}
}
