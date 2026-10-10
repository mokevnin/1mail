// Package contactexport is the subject-access Data export of one Contact
// (GDPR Art. 15/20, ADR 0021): a streamed JSON document holding the Contact, its
// Tags, Visitors, all Events, the Unsubscribe/Suppression/Confirmation rows for
// its Destinations and the OutboundMessage / BroadcastRecipient delivery
// metadata. Rendered message bodies are never part of it (no entity read here
// carries one). Large collections are read in keyset pages and written straight to
// the writer, so the document is never buffered whole.
package contactexport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/samber/lo"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/confirmation"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/ent/tag"
	"github.com/mokevnin/1mail/ent/unsubscribe"
	"github.com/mokevnin/1mail/ent/visitor"
)

// pageSize bounds how many rows of a large collection are held at once.
const pageSize = 500

// ErrNotFound means no Contact in the Workspace matches the identifier.
var ErrNotFound = errors.New("contactexport: contact not found")

// ErrIdentifier means the request named neither or both of id and email.
var ErrIdentifier = errors.New("contactexport: exactly one of id or email is required")

// Find resolves the Contact to export by exactly one of its id or email, inside
// the Workspace of s: another Workspace's Contact is not found.
func Find(ctx context.Context, s *ent.Scoped, id *int64, email *string) (*ent.Contact, error) {
	if (id == nil) == (email == nil) {
		return nil, ErrIdentifier
	}
	q := s.Contact().Query()
	if id != nil {
		q = q.Where(contact.ID(*id))
	} else {
		q = q.Where(contact.Email(strings.ToLower(strings.TrimSpace(*email))))
	}
	c, err := q.Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return c, err
}

// Stream writes the export of c to w as one JSON object. The Contact and every
// member are read through s, so nothing outside the Workspace can appear.
func Stream(ctx context.Context, s *ent.Scoped, w io.Writer, c *ent.Contact) error {
	visitors, err := s.Visitor().Query().Where(visitor.ContactID(c.ID)).Order(ent.Asc(visitor.FieldID)).All(ctx)
	if err != nil {
		return err
	}
	tags, err := s.Tag().Query().Where(tag.HasContactsWith(contact.ID(c.ID))).Order(ent.Asc(tag.FieldID)).All(ctx)
	if err != nil {
		return err
	}

	// The Contact's Destinations (ADR 0001): its email address. Rows are matched by
	// Destination as well as by contact_id, so a transactional send to the address
	// that never had a Contact is included.
	var destinations []string
	if c.Email != nil {
		destinations = append(destinations, strings.ToLower(*c.Email))
	}
	visitorIDs := lo.Map(visitors, func(v *ent.Visitor, _ int) string { return v.VisitorID })

	unsubscribes, err := s.Unsubscribe().Query().
		Where(unsubscribe.Or(unsubscribe.ContactID(c.ID), unsubscribe.DestinationIn(destinations...))).
		Order(ent.Asc(unsubscribe.FieldID)).All(ctx)
	if err != nil {
		return err
	}
	suppressions, err := s.Suppression().Query().
		Where(suppression.Or(suppression.ContactID(c.ID), suppression.DestinationIn(destinations...))).
		Order(ent.Asc(suppression.FieldID)).All(ctx)
	if err != nil {
		return err
	}
	confirmations, err := s.Confirmation().Query().
		Where(confirmation.Or(confirmation.ContactID(c.ID), confirmation.DestinationIn(destinations...))).
		Order(ent.Asc(confirmation.FieldID)).All(ctx)
	if err != nil {
		return err
	}

	sw := &streamWriter{w: w}
	sw.raw(`{"contact":`)
	sw.value(c)
	sw.raw(`,"tags":`)
	sw.value(nonNil(tags))
	sw.raw(`,"visitors":`)
	sw.value(nonNil(visitors))
	sw.raw(`,"unsubscribes":`)
	sw.value(nonNil(unsubscribes))
	sw.raw(`,"suppressions":`)
	sw.value(nonNil(suppressions))
	sw.raw(`,"confirmations":`)
	sw.value(nonNil(confirmations))

	// Events are matched by contact_id and by the Contact's visitor ids (ADR 0021);
	// an empty visitor list matches nothing.
	sw.raw(`,"events":`)
	err = streamPages(ctx, sw, func(after int64) ([]*ent.Event, error) {
		return s.Event().Query().
			Where(event.Or(event.ContactID(c.ID), event.VisitorIDIn(visitorIDs...)), event.IDGT(after)).
			Order(ent.Asc(event.FieldID)).Limit(pageSize).All(ctx)
	}, func(e *ent.Event) int64 { return e.ID })
	if err != nil {
		return err
	}

	sw.raw(`,"outbound_messages":`)
	err = streamPages(ctx, sw, func(after int64) ([]*ent.OutboundMessage, error) {
		return s.OutboundMessage().Query().
			Where(outboundmessage.Or(outboundmessage.ContactID(c.ID), outboundmessage.DestinationIn(destinations...)), outboundmessage.IDGT(after)).
			Order(ent.Asc(outboundmessage.FieldID)).Limit(pageSize).All(ctx)
	}, func(m *ent.OutboundMessage) int64 { return m.ID })
	if err != nil {
		return err
	}

	sw.raw(`,"broadcast_recipients":`)
	err = streamPages(ctx, sw, func(after int64) ([]*ent.BroadcastRecipient, error) {
		return s.BroadcastRecipient().Query().
			Where(broadcastrecipient.ContactID(c.ID), broadcastrecipient.IDGT(after)).
			Order(ent.Asc(broadcastrecipient.FieldID)).Limit(pageSize).All(ctx)
	}, func(r *ent.BroadcastRecipient) int64 { return r.ID })
	if err != nil {
		return err
	}
	sw.raw(`}`)
	return sw.err
}

// Open starts the export of c in the background and returns the stream to hand to
// the HTTP response: the document is produced as the response is read. The
// producer ends with the request's context, so an abandoned download cannot leak it.
func Open(ctx context.Context, s *ent.Scoped, c *ent.Contact) io.Reader {
	pr, pw := io.Pipe()
	go func() { pw.CloseWithError(Stream(ctx, s, pw, c)) }()
	go func() {
		<-ctx.Done()
		pr.CloseWithError(ctx.Err())
	}()
	return pr
}

// Filename is the suggested download name of c's export.
func Filename(c *ent.Contact) string {
	return "contact-" + strconv.FormatInt(c.ID, 10) + "-export.json"
}

// nonNil keeps an empty collection a JSON array rather than null.
func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}

// streamPages writes a JSON array whose elements are read page by page, ordered
// by ascending id, so at most one page is in memory.
func streamPages[T any](ctx context.Context, sw *streamWriter, page func(after int64) ([]T, error), id func(T) int64) error {
	sw.raw(`[`)
	var after int64
	first := true
	for sw.err == nil {
		rows, err := page(after)
		if err != nil {
			return err
		}
		for _, r := range rows {
			if !first {
				sw.raw(`,`)
			}
			first = false
			sw.value(r)
			after = id(r)
		}
		if len(rows) < pageSize {
			break
		}
	}
	sw.raw(`]`)
	return sw.err
}

// streamWriter remembers the first write or encode error so the document code
// stays linear.
type streamWriter struct {
	w   io.Writer
	err error
}

func (s *streamWriter) raw(text string) {
	if s.err == nil {
		_, s.err = io.WriteString(s.w, text)
	}
}

func (s *streamWriter) value(v any) {
	if s.err != nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		s.err = err
		return
	}
	_, s.err = s.w.Write(b)
}
