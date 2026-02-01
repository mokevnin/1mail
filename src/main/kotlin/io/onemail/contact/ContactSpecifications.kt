package io.onemail.contact

import org.springframework.data.jpa.domain.Specification

object ContactSpecifications {
    fun hasAccountId(accountId: Long): Specification<Contact> =
        Specification { root, _, cb ->
            cb.equal(root.get<Long>("account").get<Long>("id"), accountId)
        }

    fun hasStatus(status: ContactStatus): Specification<Contact> =
        Specification { root, _, cb -> cb.equal(root.get<ContactStatus>("status"), status) }
}
