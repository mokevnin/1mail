package io.onemail.contact

import org.springframework.data.jpa.repository.JpaRepository
import org.springframework.data.jpa.repository.JpaSpecificationExecutor

interface ContactRepository :
    JpaRepository<Contact, Long>,
    JpaSpecificationExecutor<Contact> {
    fun findByAccountIdAndEmail(
        accountId: Long,
        email: String,
    ): Contact?

    fun findByIdAndAccountId(
        id: Long,
        accountId: Long,
    ): Contact?
}
