package router

import (
	"context"
	"testing"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/features/extension"
	"google.golang.org/protobuf/proto"
)

type fakeObservatory struct {
	result  *observatory.ObservationResult
	byTag   map[string]*observatory.ObservationResult
}

func (f *fakeObservatory) Type() interface{} { return extension.ObservatoryType() }
func (f *fakeObservatory) Start() error      { return nil }
func (f *fakeObservatory) Close() error      { return nil }
func (f *fakeObservatory) GetObservation(context.Context) (proto.Message, error) {
	return f.result, nil
}

func (f *fakeObservatory) GetObservationByTag(ctx context.Context, tag string) (proto.Message, error) {
	return f.byTag[tag], nil
}

func TestLeastPingStrategyUsesTaggedObservatory(t *testing.T) {
	s := &LeastPingStrategy{
		observatoryTag: "youtube",
		observatory: &fakeObservatory{
			result: &observatory.ObservationResult{
				Status: []*observatory.OutboundStatus{
					{OutboundTag: "a", Alive: true, Delay: 1},
					{OutboundTag: "b", Alive: true, Delay: 500},
				},
			},
			byTag: map[string]*observatory.ObservationResult{
				"youtube": {
					Status: []*observatory.OutboundStatus{
						{OutboundTag: "a", Alive: true, Delay: 100},
						{OutboundTag: "b", Alive: true, Delay: 50},
					},
				},
			},
		},
	}

	if got := s.PickOutbound([]string{"a", "b"}); got != "b" {
		t.Fatalf("expected tagged observatory result to pick b, got %q", got)
	}
}

func TestRandomStrategyFallsBackToDefaultObservatory(t *testing.T) {
	s := &RandomStrategy{
		observatory: &fakeObservatory{
			result: &observatory.ObservationResult{
				Status: []*observatory.OutboundStatus{
					{OutboundTag: "dead", Alive: false},
					{OutboundTag: "alive", Alive: true},
				},
			},
		},
	}

	if got := s.PickOutbound([]string{"dead", "alive"}); got != "alive" {
		t.Fatalf("expected alive, got %q", got)
	}
}
