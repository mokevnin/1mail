package site

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/integration"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/messaging/ses"
	"github.com/mokevnin/1mail/internal/messaging/smtp"
	"github.com/mokevnin/1mail/internal/sendlimit"
	"github.com/mokevnin/1mail/internal/service"
)

// SiteIntegrationsList returns the workspace's sending-provider integrations
// with secrets redacted.
func (h *Handlers) SiteIntegrationsList(ctx context.Context, params siteapi.SiteIntegrationsListParams) (siteapi.SiteIntegrationsListRes, error) {
	s, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.Integration().Query().
		Order(ent.Asc(integration.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	usage, err := sendlimit.Usage(ctx, s, time.Now())
	if err != nil {
		return nil, err
	}
	items := make(siteapi.SiteIntegrationsListOKApplicationJSON, len(rows))
	for i, row := range rows {
		res, err := h.integrationToResource(row, usage)
		if err != nil {
			return nil, err
		}
		items[i] = res
	}
	return &items, nil
}

// SiteIntegrationsCreate stores a new provider integration; credentials are
// encrypted at rest and never returned.
func (h *Handlers) SiteIntegrationsCreate(ctx context.Context, req *siteapi.SiteCreateIntegrationInput, params siteapi.SiteIntegrationsCreateParams) (siteapi.SiteIntegrationsCreateRes, error) {
	s, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteIntegrationsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		v := siteapi.SiteIntegrationsCreateUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity, i18n.T("errors.name_empty", nil),
			map[string][]string{"name": {i18n.T("errors.name_empty", nil)}},
		))
		return &v, nil
	}

	provider, channel, plaintext, verr := h.encodeConfigInput(req.Config)
	if verr != nil {
		v := siteapi.SiteIntegrationsCreateUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity, verr.Error(),
			map[string][]string{"config": {verr.Error()}},
		))
		return &v, nil
	}

	encrypted, err := h.cipher.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	maxPerSecond, maxPerDay := limitValue(req.MaxPerSecond), limitValue(req.MaxPerDay)

	enabled := req.Enabled.Or(true)
	isDefault := req.IsDefault.Or(false)

	row, err := h.createIntegration(ctx, s, integrationDraft{
		name: name, channel: channel, provider: provider, encrypted: encrypted,
		enabled: enabled, isDefault: isDefault, maxPerSecond: maxPerSecond, maxPerDay: maxPerDay,
	})
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteIntegrationsCreateConflict(problem(http.StatusConflict, "a default provider already exists for this channel"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	res, err := h.integrationResource(ctx, s, row)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// SiteIntegrationsGet returns one integration with secrets redacted.
func (h *Handlers) SiteIntegrationsGet(ctx context.Context, params siteapi.SiteIntegrationsGetParams) (siteapi.SiteIntegrationsGetRes, error) {
	s, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteIntegrationsGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteIntegrationsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	row, err := s.Integration().Get(ctx, id)
	if ent.IsNotFound(err) {
		v := siteapi.SiteIntegrationsGetNotFound(problem(http.StatusNotFound, "integration not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := h.integrationResource(ctx, s, row)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// SiteIntegrationsUpdate edits an integration. Omitting config keeps the stored
// credentials; a supplied config must match the integration's provider kind.
// Within a supplied config, blank secret fields (password, secret access key)
// keep the stored secret rather than clearing it, since secrets are never echoed
// back on read — so a partial edit cannot accidentally wipe a credential.
func (h *Handlers) SiteIntegrationsUpdate(ctx context.Context, req *siteapi.SiteUpdateIntegrationInput, params siteapi.SiteIntegrationsUpdateParams) (siteapi.SiteIntegrationsUpdateRes, error) {
	s, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteIntegrationsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteIntegrationsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	row, err := s.Integration().Get(ctx, id)
	if ent.IsNotFound(err) {
		v := siteapi.SiteIntegrationsUpdateNotFound(problem(http.StatusNotFound, "integration not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	// Validate and collect the requested changes up front so 422s short-circuit
	// before we open a transaction. The writes themselves happen inside withTx.
	var (
		setName      *string
		setEnabled   *bool
		setEncrypted *string
	)
	if v, ok := req.Name.Get(); ok {
		name := strings.TrimSpace(v)
		if name == "" {
			r := siteapi.SiteIntegrationsUpdateUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity, i18n.T("errors.name_empty", nil),
				map[string][]string{"name": {i18n.T("errors.name_empty", nil)}},
			))
			return &r, nil
		}
		setName = &name
	}
	if v, ok := req.Enabled.Get(); ok {
		setEnabled = &v
	}
	perSecond, perDay := limitValue(req.MaxPerSecond), limitValue(req.MaxPerDay)

	if cfg, ok := req.Config.Get(); ok {
		provider, _, plaintext, verr := h.encodeConfigInput(cfg)
		if verr != nil {
			r := siteapi.SiteIntegrationsUpdateUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity, verr.Error(),
				map[string][]string{"config": {verr.Error()}},
			))
			return &r, nil
		}
		if string(provider) != row.Provider.String() {
			r := siteapi.SiteIntegrationsUpdateUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity, "config kind must match the integration provider",
				map[string][]string{"config": {i18n.T("errors.integration_kind_mismatch", nil)}},
			))
			return &r, nil
		}
		// Secrets are redacted on read, so a partial edit re-sends the config with
		// blank secret fields. Carry the stored secrets forward (empty == keep)
		// instead of overwriting them with nothing.
		prev, err := h.cipher.Decrypt(row.ConfigEncrypted)
		if err != nil {
			return nil, err
		}
		plaintext, err = mergeSecrets(provider, plaintext, prev)
		if err != nil {
			return nil, err
		}
		encrypted, err := h.cipher.Encrypt(plaintext)
		if err != nil {
			return nil, err
		}
		setEncrypted = &encrypted
	}

	// A default toggle touches sibling rows, so promotion clears the channel's
	// current default first; both must be in the same tx as the update itself.
	promote := false
	var setDefault *bool
	if v, ok := req.IsDefault.Get(); ok {
		if v && !row.IsDefault {
			promote = true
		} else {
			setDefault = &v
		}
	}

	var updated *ent.Integration
	err = h.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, _ events.Publisher) error {
		if promote {
			if err := clearDefault(ctx, ts, integration.Channel(row.Channel.String())); err != nil {
				return err
			}
		}
		upd := ts.Integration().UpdateOneID(row.ID)
		if setName != nil {
			upd.SetName(*setName)
		}
		if setEnabled != nil {
			upd.SetEnabled(*setEnabled)
		}
		if setEncrypted != nil {
			upd.SetConfigEncrypted(*setEncrypted)
		}
		switch {
		case perSecond != nil:
			upd.SetMaxPerSecond(*perSecond)
		case req.MaxPerSecond.IsSet(): // explicit null
			upd.ClearMaxPerSecond()
		}
		switch {
		case perDay != nil:
			upd.SetMaxPerDay(*perDay)
		case req.MaxPerDay.IsSet():
			upd.ClearMaxPerDay()
		}
		if promote {
			upd.SetIsDefault(true)
		} else if setDefault != nil {
			upd.SetIsDefault(*setDefault)
		}
		var err error
		updated, err = upd.Save(ctx)
		return err
	})
	if service.IsUniqueViolation(err) {
		r := siteapi.SiteIntegrationsUpdateConflict(problem(http.StatusConflict, "a default provider already exists for this channel"))
		return &r, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := h.integrationResource(ctx, s, updated)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// SiteIntegrationsDelete removes an integration.
func (h *Handlers) SiteIntegrationsDelete(ctx context.Context, params siteapi.SiteIntegrationsDeleteParams) (siteapi.SiteIntegrationsDeleteRes, error) {
	s, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteIntegrationsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteIntegrationsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	n, err := s.Integration().Delete().
		Where(integration.IDEQ(id)).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		v := siteapi.SiteIntegrationsDeleteNotFound(problem(http.StatusNotFound, "integration not found"))
		return &v, nil
	}
	return &siteapi.SiteIntegrationsDeleteNoContent{}, nil
}

// --- helpers ---

// integrationDraft is the validated input of a new Integration.
type integrationDraft struct {
	name         string
	channel      integration.Channel
	provider     integration.Provider
	encrypted    string
	enabled      bool
	isDefault    bool
	maxPerSecond *int
	maxPerDay    *int
}

// limitValue reads one Send rate limit from its request field: null (and, on create,
// omitted) means no limit and yields a nil pointer. The contract bounds a set value.
func limitValue[T ~int32](in interface{ Get() (T, bool) }) *int {
	v, ok := in.Get()
	if !ok {
		return nil
	}
	n := int(v)
	return &n
}

// createIntegration inserts a row in a tx, clearing any sibling default first
// when this one is the new default (so the partial unique index never trips on
// our own writes).
func (h *Handlers) createIntegration(ctx context.Context, s *ent.Scoped, d integrationDraft) (*ent.Integration, error) {
	var row *ent.Integration
	err := h.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, _ events.Publisher) error {
		if d.isDefault {
			if err := clearDefault(ctx, ts, d.channel); err != nil {
				return err
			}
		}
		var err error
		row, err = ts.Integration().Create().
			SetName(d.name).
			SetChannel(d.channel).
			SetProvider(d.provider).
			SetConfigEncrypted(d.encrypted).
			SetEnabled(d.enabled).
			SetIsDefault(d.isDefault).
			SetNillableMaxPerSecond(d.maxPerSecond).
			SetNillableMaxPerDay(d.maxPerDay).
			Save(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return row, nil
}

// clearDefault unsets the existing default integration for (workspace, channel).
func clearDefault(ctx context.Context, ts *ent.Scoped, channel integration.Channel) error {
	_, err := ts.Integration().Update().
		Where(
			integration.ChannelEQ(channel),
			integration.IsDefault(true),
		).
		SetIsDefault(false).
		Save(ctx)
	return err
}

// encodeConfigInput validates the typed config union, derives the
// provider/channel and returns the cleartext JSON to encrypt.
func (h *Handlers) encodeConfigInput(in siteapi.SiteIntegrationConfigInput) (integration.Provider, integration.Channel, []byte, error) {
	var (
		provider  integration.Provider
		plaintext []byte
		err       error
	)
	switch in.OneOf.Type {
	case siteapi.SiteSmtpConfigInputSiteIntegrationConfigInputSum:
		c := in.OneOf.SiteSmtpConfigInput
		provider = integration.ProviderSMTP
		plaintext, err = json.Marshal(smtp.Config{
			Host:     c.Host,
			Port:     int(c.Port),
			Username: c.Username.Or(""),
			Password: c.Password.Or(""),
			From:     string(c.From),
			FromName: c.FromName.Or(""),
		})
	case siteapi.SiteSesConfigInputSiteIntegrationConfigInputSum:
		c := in.OneOf.SiteSesConfigInput
		provider = integration.ProviderSes
		plaintext, err = json.Marshal(ses.Config{
			Region:          c.Region,
			AccessKeyID:     c.AccessKeyId,
			SecretAccessKey: c.SecretAccessKey,
			From:            string(c.From),
			FromName:        c.FromName.Or(""),
			Endpoint:        c.Endpoint.Or(""),
		})
	default:
		return "", "", nil, errUnknownProvider
	}
	if err != nil {
		return "", "", nil, err
	}

	channel, ok := h.catalog.ChannelOf(messaging.Provider(provider))
	if !ok {
		return "", "", nil, errUnknownProvider
	}
	if err := h.catalog.Validate(channel, messaging.Provider(provider), plaintext); err != nil {
		return "", "", nil, err
	}
	return provider, integration.Channel(channel), plaintext, nil
}

var errUnknownProvider = errors.New("unsupported provider")

// mergeSecrets carries forward stored secrets that the client did not resupply.
// Secrets are redacted on read, so a config update re-sends blank secret fields;
// an empty secret therefore means "keep the stored value", not "clear it". The
// provider kind is guaranteed to match (validated before this is called).
func mergeSecrets(provider integration.Provider, next, prev []byte) ([]byte, error) {
	switch provider {
	case integration.ProviderSMTP:
		var n, p smtp.Config
		if err := json.Unmarshal(next, &n); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(prev, &p); err != nil {
			return nil, err
		}
		if n.Password == "" {
			n.Password = p.Password
		}
		return json.Marshal(n)
	case integration.ProviderSes:
		var n, p ses.Config
		if err := json.Unmarshal(next, &n); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(prev, &p); err != nil {
			return nil, err
		}
		if n.SecretAccessKey == "" {
			n.SecretAccessKey = p.SecretAccessKey
		}
		return json.Marshal(n)
	}
	return next, nil
}

// integrationToResource decrypts the stored config and builds the API resource
// with all secrets redacted.
func (h *Handlers) integrationToResource(row *ent.Integration, usage map[int64]int) (siteapi.SiteIntegrationResource, error) {
	res := siteapi.SiteIntegrationResource{
		ID:           siteapi.EntityId(strconv.FormatInt(row.ID, 10)),
		Name:         row.Name,
		Channel:      siteapi.SiteIntegrationChannel(row.Channel.String()),
		Provider:     siteapi.SiteIntegrationProvider(row.Provider.String()),
		Enabled:      row.Enabled,
		IsDefault:    row.IsDefault,
		MaxPerSecond: limitOpt(row.MaxPerSecond),
		MaxPerDay:    limitOpt(row.MaxPerDay),
		SendLimit:    sendLimitStatus(row, usage),
		CreatedAt:    siteapi.Timestamp(row.CreatedAt),
		UpdatedAt:    siteapi.Timestamp(row.UpdatedAt),
	}

	plaintext, err := h.cipher.Decrypt(row.ConfigEncrypted)
	if err != nil {
		return res, err
	}

	switch row.Provider {
	case integration.ProviderSMTP:
		var c smtp.Config
		if err := json.Unmarshal(plaintext, &c); err != nil {
			return res, err
		}
		out := siteapi.SiteSmtpConfig{
			Kind:     siteapi.SiteSmtpConfigKindSMTP,
			Host:     c.Host,
			Port:     int32(c.Port),
			From:     siteapi.EmailAddress(c.From),
			Username: optNilString(c.Username),
			FromName: optNilString(c.FromName),
		}
		res.Config = siteapi.SiteIntegrationConfig{OneOf: siteapi.NewSiteSmtpConfigSiteIntegrationConfigSum(out)}
	case integration.ProviderSes:
		var c ses.Config
		if err := json.Unmarshal(plaintext, &c); err != nil {
			return res, err
		}
		out := siteapi.SiteSesConfig{
			Kind:     siteapi.SiteSesConfigKindSes,
			Region:   c.Region,
			From:     siteapi.EmailAddress(c.From),
			FromName: optNilString(c.FromName),
			Endpoint: optNilString(c.Endpoint),
		}
		if n := len(c.AccessKeyID); n > 0 {
			last4 := c.AccessKeyID
			if n > 4 {
				last4 = c.AccessKeyID[n-4:]
			}
			out.AccessKeyIdLast4 = siteapi.NewOptNilString(last4)
		}
		res.Config = siteapi.SiteIntegrationConfig{OneOf: siteapi.NewSiteSesConfigSiteIntegrationConfigSum(out)}
	default:
		return res, errUnknownProvider
	}
	return res, nil
}

// limitOpt renders a stored limit: null when there is none.
func limitOpt(v *int) siteapi.NilInt32 {
	if v == nil {
		return siteapi.NilInt32{Null: true}
	}
	return siteapi.NewNilInt32(int32(*v))
}

// optNilString returns a set OptNilString for non-empty input, else the zero
// (unset) value — so redacted optional fields are omitted rather than blank.
func optNilString(s string) siteapi.OptNilString {
	if s == "" {
		return siteapi.OptNilString{}
	}
	return siteapi.NewOptNilString(s)
}

// integrationResource renders one Integration with its 24-hour usage, for the
// single-row responses (get, create, update).
func (h *Handlers) integrationResource(ctx context.Context, s *ent.Scoped, row *ent.Integration) (siteapi.SiteIntegrationResource, error) {
	usage, err := sendlimit.Usage(ctx, s, time.Now())
	if err != nil {
		return siteapi.SiteIntegrationResource{}, err
	}
	return h.integrationToResource(row, usage)
}

// sendLimitStatus renders an Integration's enforced Send rate limit with its sent
// count over the last 24 hours (from the per-Integration usage map).
func sendLimitStatus(row *ent.Integration, usage map[int64]int) siteapi.SiteSendLimitStatus {
	eff := sendlimit.EffectiveOf(row)
	out := siteapi.SiteSendLimitStatus{
		PerSecond:   sendLimitValue(eff.PerSecond),
		PerDay:      sendLimitValue(eff.PerDay),
		SentLast24h: int32(usage[row.ID]),
		Warnings:    []siteapi.SiteSendLimitWarning{},
	}
	if eff.Unlimited() {
		out.Warnings = append(out.Warnings, siteapi.SiteSendLimitWarningUnlimited)
	}
	return out
}

func sendLimitValue(v sendlimit.Value) siteapi.SiteSendLimitValue {
	out := siteapi.SiteSendLimitValue{Limit: limitOpt(v.Limit)}
	if v.Source == "" {
		out.Source = siteapi.NilSiteSendLimitSource{Null: true}
	} else {
		out.Source = siteapi.NewNilSiteSendLimitSource(siteapi.SiteSendLimitSource(v.Source))
	}
	return out
}
