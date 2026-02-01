package io.onemail.segment

import org.springframework.data.jpa.domain.Specification

object SegmentSpecifications {
    fun hasAccountId(accountId: Long): Specification<Segment> =
        Specification { root, _, cb ->
            cb.equal(root.get<Long>("account").get<Long>("id"), accountId)
        }
}
