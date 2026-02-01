package io.onemail.broadcast

import org.springframework.data.jpa.domain.Specification

object BroadcastSpecifications {
    fun hasAccountId(accountId: Long): Specification<Broadcast> =
        Specification { root, _, cb ->
            cb.equal(root.get<Long>("account").get<Long>("id"), accountId)
        }
}
