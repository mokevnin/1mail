package external

import (
	"context"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/problems"
)

func (h *Handlers) BroadcastsList(ctx context.Context, req externalapi.BroadcastsListRequestObject) (externalapi.BroadcastsListResponseObject, error) {
	return externalapi.BroadcastsList401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) BroadcastsCreate(ctx context.Context, req externalapi.BroadcastsCreateRequestObject) (externalapi.BroadcastsCreateResponseObject, error) {
	return externalapi.BroadcastsCreate401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) BroadcastsGet(ctx context.Context, req externalapi.BroadcastsGetRequestObject) (externalapi.BroadcastsGetResponseObject, error) {
	return externalapi.BroadcastsGet401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) BroadcastsUpdate(ctx context.Context, req externalapi.BroadcastsUpdateRequestObject) (externalapi.BroadcastsUpdateResponseObject, error) {
	return externalapi.BroadcastsUpdate401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}

func (h *Handlers) BroadcastsDelete(ctx context.Context, req externalapi.BroadcastsDeleteRequestObject) (externalapi.BroadcastsDeleteResponseObject, error) {
	return externalapi.BroadcastsDelete401ApplicationProblemPlusJSONResponse(problems.NotImplemented("not implemented").External()), nil
}
