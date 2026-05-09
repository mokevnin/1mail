package external

import (
	"context"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/problems"
)

func (h *Handlers) SegmentsList(ctx context.Context, req externalapi.SegmentsListRequestObject) (externalapi.SegmentsListResponseObject, error) {
	return externalapi.SegmentsList401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) SegmentsCreate(ctx context.Context, req externalapi.SegmentsCreateRequestObject) (externalapi.SegmentsCreateResponseObject, error) {
	return externalapi.SegmentsCreate401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) SegmentsGet(ctx context.Context, req externalapi.SegmentsGetRequestObject) (externalapi.SegmentsGetResponseObject, error) {
	return externalapi.SegmentsGet401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) SegmentsUpdate(ctx context.Context, req externalapi.SegmentsUpdateRequestObject) (externalapi.SegmentsUpdateResponseObject, error) {
	return externalapi.SegmentsUpdate401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) SegmentsDelete(ctx context.Context, req externalapi.SegmentsDeleteRequestObject) (externalapi.SegmentsDeleteResponseObject, error) {
	return externalapi.SegmentsDelete401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}
