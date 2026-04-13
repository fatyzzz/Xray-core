package router

import (
	"context"

	"github.com/xtls/xray-core/app/observatory"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/features/extension"
	"google.golang.org/protobuf/proto"
)

type taggedObservatory interface {
	GetObservationByTag(ctx context.Context, tag string) (proto.Message, error)
}

func getObservationResult(
	ctx context.Context,
	observer extension.Observatory,
	tag string,
) (*observatory.ObservationResult, error) {
	if tag != "" {
		if observer == nil {
			return nil, errors.New("observer is nil")
		}
		tagged, ok := observer.(taggedObservatory)
		if !ok {
			return nil, errors.New("observer does not support observatory tags")
		}
		msg, err := tagged.GetObservationByTag(ctx, tag)
		if err != nil {
			return nil, err
		}
		result, ok := msg.(*observatory.ObservationResult)
		if !ok {
			return nil, errors.New("unexpected observatory result type")
		}
		return result, nil
	}

	if observer == nil {
		return nil, errors.New("observer is nil")
	}

	msg, err := observer.GetObservation(ctx)
	if err != nil {
		return nil, err
	}
	result, ok := msg.(*observatory.ObservationResult)
	if !ok {
		return nil, errors.New("unexpected observation result type")
	}
	return result, nil
}
