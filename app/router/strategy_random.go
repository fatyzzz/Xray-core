package router

import (
	"context"

	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/dice"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/extension"
)

// RandomStrategy represents a random balancing strategy
type RandomStrategy struct {
	FallbackTag string

	ctx            context.Context
	observatory    extension.Observatory
	observatoryTag string
}

func (s *RandomStrategy) InjectContext(ctx context.Context) {
	s.ctx = ctx
	if len(s.FallbackTag) > 0 || s.observatoryTag != "" {
		common.Must(core.RequireFeatures(s.ctx, func(observatory extension.Observatory) error {
			s.observatory = observatory
			return nil
		}))
	}
}

func (s *RandomStrategy) GetPrincipleTarget(strings []string) []string {
	return strings
}

func (s *RandomStrategy) PickOutbound(candidates []string) string {
	if s.observatory != nil {
		observeResult, err := getObservationResult(s.ctx, s.observatory, s.observatoryTag)
		if err == nil {
			aliveTags := make([]string, 0)
			statusMap := make(map[string]bool, len(observeResult.Status))
			for _, outboundStatus := range observeResult.Status {
				statusMap[outboundStatus.OutboundTag] = outboundStatus.Alive
			}
			for _, candidate := range candidates {
				if alive, found := statusMap[candidate]; found {
					if alive {
						aliveTags = append(aliveTags, candidate)
					}
				} else {
					// unfound candidate is considered alive
					aliveTags = append(aliveTags, candidate)
				}
			}
			candidates = aliveTags
		}
	}

	count := len(candidates)
	if count == 0 {
		// goes to fallbackTag
		return ""
	}
	return candidates[dice.Roll(count)]
}
