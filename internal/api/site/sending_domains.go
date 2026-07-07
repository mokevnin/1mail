package site

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/sendingdomain"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/sending"
	"github.com/mokevnin/1mail/internal/service"
)

// defaultDKIMSelector is used when the client does not supply one.
const defaultDKIMSelector = "1mail"

// sendingDomainResource builds the API resource, computing the DNS records the
// user publishes. Built by hand (not goverter) because the records are derived,
// and the private key is never exposed.
func sendingDomainResource(d *ent.SendingDomain) siteapi.SiteSendingDomainResource {
	dkimHost, dkimValue := sending.DKIMRecord(d.DkimSelector, d.Domain, d.DkimPublicKey)
	spfHost, spfValue := sending.SPFRecord(d.Domain)
	dmarcHost, dmarcValue := sending.DMARCRecord(d.Domain)

	res := siteapi.SiteSendingDomainResource{
		ID:           siteapi.EntityId(strconv.FormatInt(d.ID, 10)),
		Domain:       d.Domain,
		DkimSelector: d.DkimSelector,
		Verified:     d.Verified,
		DkimRecord:   dnsRecord(dkimHost, dkimValue),
		SpfRecord:    dnsRecord(spfHost, spfValue),
		DmarcRecord:  dnsRecord(dmarcHost, dmarcValue),
		CreatedAt:    siteapi.Timestamp(d.CreatedAt),
		UpdatedAt:    siteapi.Timestamp(d.UpdatedAt),
	}
	if d.LastCheckedAt != nil {
		res.LastCheckedAt = siteapi.NewOptNilTimestamp(siteapi.Timestamp(*d.LastCheckedAt))
	}
	if d.VerifiedAt != nil {
		res.VerifiedAt = siteapi.NewOptNilTimestamp(siteapi.Timestamp(*d.VerifiedAt))
	}
	return res
}

func dnsRecord(host, value string) siteapi.SiteDnsRecord {
	return siteapi.SiteDnsRecord{Type: siteapi.SiteDnsRecordTypeTXT, Host: host, Value: value}
}

func (h *Handlers) SiteSendingDomainsList(ctx context.Context, params siteapi.SiteSendingDomainsListParams) (siteapi.SiteSendingDomainsListRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsListNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	var pagePtr, pageSizePtr *int32
	if v, ok := params.Page.Get(); ok {
		pagePtr = &v
	}
	if v, ok := params.PageSize.Get(); ok {
		pageSizePtr = &v
	}
	page, pageSize := pagination.Normalize(pagePtr, pageSizePtr)

	q := h.ent.SendingDomain.Query().Where(sendingdomain.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Desc(sendingdomain.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteSendingDomainResource, len(items))
	for i, d := range items {
		resources[i] = sendingDomainResource(d)
	}
	return &siteapi.SiteSendingDomainsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteSendingDomainsCreate(ctx context.Context, req *siteapi.SiteCreateSendingDomainInput, params siteapi.SiteSendingDomainsCreateParams) (siteapi.SiteSendingDomainsCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	domain := normalizeDomain(req.Domain)
	if !validDomain(domain) {
		v := siteapi.SiteSendingDomainsCreateUnprocessableEntity(problemWithErrors(http.StatusUnprocessableEntity, i18n.T("errors.validation_failed", nil), map[string][]string{
			"domain": {i18n.T("errors.sending_domain_invalid", nil)},
		}))
		return &v, nil
	}
	selector := defaultDKIMSelector
	if v, ok := req.DkimSelector.Get(); ok && strings.TrimSpace(v) != "" {
		selector = strings.TrimSpace(v)
	}

	privPEM, pubTXT, err := sending.GenerateKeypair()
	if err != nil {
		return nil, err
	}
	encrypted, err := h.cipher.Encrypt(privPEM)
	if err != nil {
		return nil, err
	}

	// Wrap the insert in a savepoint so a unique-violation (duplicate domain)
	// rolls back only this write, not the caller's surrounding transaction.
	var d *ent.SendingDomain
	err = h.withTx(ctx, func(tx *ent.Tx) error {
		var cerr error
		d, cerr = tx.SendingDomain.Create().
			SetWorkspaceID(ws).
			SetDomain(domain).
			SetDkimSelector(selector).
			SetDkimPrivateKeyEncrypted(encrypted).
			SetDkimPublicKey(pubTXT).
			Save(ctx)
		return cerr
	})
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteSendingDomainsCreateConflict(problem(http.StatusConflict, "this domain is already added"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := sendingDomainResource(d)
	return &res, nil
}

func (h *Handlers) SiteSendingDomainsGet(ctx context.Context, params siteapi.SiteSendingDomainsGetParams) (siteapi.SiteSendingDomainsGetRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSendingDomainsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	d, err := h.ent.SendingDomain.Query().
		Where(sendingdomain.IDEQ(id), sendingdomain.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsGetNotFound(problem(http.StatusNotFound, "sending domain not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := sendingDomainResource(d)
	return &res, nil
}

func (h *Handlers) SiteSendingDomainsDelete(ctx context.Context, params siteapi.SiteSendingDomainsDeleteParams) (siteapi.SiteSendingDomainsDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSendingDomainsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.SendingDomain.DeleteOneID(id).Where(sendingdomain.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsDeleteNotFound(problem(http.StatusNotFound, "sending domain not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteSendingDomainsDeleteNoContent{}, nil
}

func (h *Handlers) SiteSendingDomainsVerify(ctx context.Context, params siteapi.SiteSendingDomainsVerifyParams) (siteapi.SiteSendingDomainsVerifyRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSendingDomainsVerifyNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSendingDomainsVerifyBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	// Confirm the domain exists in this workspace before enqueuing.
	exists, err := h.ent.SendingDomain.Query().
		Where(sendingdomain.IDEQ(id), sendingdomain.WorkspaceID(ws)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		v := siteapi.SiteSendingDomainsVerifyNotFound(problem(http.StatusNotFound, "sending domain not found"))
		return &v, nil
	}
	if err := h.domainVerify.EnqueueSendingDomainVerify(ctx, id); err != nil {
		return nil, err
	}
	return &siteapi.SiteSendingDomainsVerifyNoContent{}, nil
}

// normalizeDomain lowercases and trims a domain and strips a trailing dot.
func normalizeDomain(raw string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
}

// validDomain is a lightweight sanity check (not a full RFC 1035 validator): at
// least one dot, no scheme/spaces, plausible label characters.
func validDomain(d string) bool {
	if d == "" || len(d) > 253 || strings.ContainsAny(d, " /:@") || !strings.Contains(d, ".") {
		return false
	}
	for _, label := range strings.Split(d, ".") {
		if label == "" {
			return false
		}
	}
	return true
}
