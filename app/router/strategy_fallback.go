package router

import (
	"context"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/extension"
)

// FallbackStrategy picks the first usable outbound in selector order.
type FallbackStrategy struct {
	ctx         context.Context
	observatory extension.Observatory
}

func (s *FallbackStrategy) InjectContext(ctx context.Context) {
	s.ctx = ctx

	if instance := core.FromContext(ctx); instance != nil {
		common.Must(instance.RequireFeatures(func(observatory extension.Observatory) {
			s.observatory = observatory
		}, true))
	}
}

func (s *FallbackStrategy) GetPrincipleTarget(tags []string) []string {
	if tag := s.PickOutbound(tags); tag != "" {
		return []string{tag}
	}
	return nil
}

func (s *FallbackStrategy) PickOutbound(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}
	if s.observatory == nil {
		return candidates[0]
	}

	observeReport, err := s.observatory.GetObservation(s.ctx)
	if err != nil {
		return candidates[0]
	}
	result, ok := observeReport.(*observatory.ObservationResult)
	if !ok {
		return candidates[0]
	}

	statusMap := make(map[string]*observatory.OutboundStatus, len(result.Status))
	for _, outboundStatus := range result.Status {
		statusMap[outboundStatus.OutboundTag] = outboundStatus
	}

	for _, candidate := range candidates {
		if outboundStatus, found := statusMap[candidate]; found {
			if outboundStatus.Alive {
				return candidate
			}
			continue
		}
		return candidate
	}

	return ""
}
