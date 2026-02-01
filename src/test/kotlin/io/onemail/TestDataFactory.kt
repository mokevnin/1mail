package io.onemail

import io.onemail.account.Account
import io.onemail.broadcast.Broadcast
import io.onemail.broadcast.BroadcastStatus
import io.onemail.contact.Contact
import io.onemail.contact.ContactStatus
import io.onemail.model.CreateBroadcastRequest
import io.onemail.model.CreateContactRequest
import io.onemail.model.CreateSegmentRequest
import io.onemail.model.UpdateSegmentRequest
import io.onemail.segment.Segment
import io.onemail.segment.SegmentType
import org.instancio.Instancio
import org.instancio.Select

object TestDataFactory {
    fun account(): Account =
        Instancio
            .of(Account::class.java)
            .ignore(KSelect.field(Account::id))
            .create()

    fun contact(account: Account): Contact =
        Instancio
            .of(Contact::class.java)
            .ignore(KSelect.field(Contact::id))
            .set(KSelect.field(Contact::account), account)
            .set(KSelect.field(Contact::status), ContactStatus.ACTIVE)
            .create()

    fun segment(
        account: Account,
        type: SegmentType = SegmentType.SNAPSHOT,
    ): Segment =
        Instancio
            .of(Segment::class.java)
            .ignore(KSelect.field(Segment::id))
            .set(KSelect.field(Segment::account), account)
            .set(KSelect.field(Segment::type), type)
            .set(KSelect.field(Segment::definition), if (type == SegmentType.RULE) "status == 'active'" else null)
            .create()

    fun broadcast(
        account: Account,
        segment: Segment,
    ): Broadcast =
        Instancio
            .of(Broadcast::class.java)
            .ignore(KSelect.field(Broadcast::id))
            .set(KSelect.field(Broadcast::account), account)
            .set(KSelect.field(Broadcast::segment), segment)
            .set(KSelect.field(Broadcast::status), BroadcastStatus.DRAFT)
            .create()

    fun createContactRequest(): CreateContactRequest =
        Instancio
            .of(CreateContactRequest::class.java)
            .set(Select.field(CreateContactRequest::class.java, "email"), "test-${System.nanoTime()}@example.com")
            .create()

    fun createSegmentRequest(type: io.onemail.model.SegmentType = io.onemail.model.SegmentType.rule): CreateSegmentRequest =
        Instancio
            .of(CreateSegmentRequest::class.java)
            .set(Select.field(CreateSegmentRequest::class.java, "type"), type)
            .set(
                Select.field(CreateSegmentRequest::class.java, "definition"),
                if (type ==
                    io.onemail.model.SegmentType.rule
                ) {
                    "status == 'active'"
                } else {
                    null
                },
            ).create()

    fun updateSegmentRequest(): UpdateSegmentRequest =
        Instancio
            .of(UpdateSegmentRequest::class.java)
            .set(
                Select.field(UpdateSegmentRequest::class.java, "definition"),
                jsonNullableString("status == 'unsubscribed'"),
            ).create()

    fun createBroadcastRequest(segmentId: Long): CreateBroadcastRequest =
        Instancio
            .of(CreateBroadcastRequest::class.java)
            .set(Select.field(CreateBroadcastRequest::class.java, "segmentId"), segmentId)
            .set(Select.field(CreateBroadcastRequest::class.java, "status"), io.onemail.model.BroadcastStatus.draft)
            .create()

    private fun jsonNullableString(value: String): org.openapitools.jackson.nullable.JsonNullable<String> =
        org.openapitools.jackson.nullable.JsonNullable
            .of(value)
}
