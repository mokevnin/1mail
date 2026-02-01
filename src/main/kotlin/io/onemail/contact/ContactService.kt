package io.onemail.contact

import io.onemail.account.Account
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.stereotype.Service
import org.springframework.transaction.annotation.Transactional

@Service
class ContactService(
    private val contactRepository: ContactRepository,
) {
    @Transactional
    fun create(
        account: Account,
        contact: Contact,
    ): Contact {
        if (contact.account.id == null) {
            contact.account = account
        }
        return contactRepository.save(contact)
    }

    fun list(
        account: Account,
        pageable: Pageable,
        status: ContactStatus?,
    ): Page<Contact> {
        val spec =
            ContactSpecifications
                .hasAccountId(account.id!!)
                .let { base ->
                    status?.let { base.and(ContactSpecifications.hasStatus(it)) } ?: base
                }
        return contactRepository.findAll(spec, pageable)
    }

    fun get(
        account: Account,
        id: Long,
    ): Contact? = contactRepository.findByIdAndAccountId(id, account.id!!)

    @Transactional
    fun update(
        account: Account,
        id: Long,
        contact: Contact,
    ): Contact? {
        val existing = contactRepository.findByIdAndAccountId(id, account.id!!) ?: return null
        val updated = mergeUpdate(existing, contact)
        return contactRepository.save(updated)
    }

    @Transactional
    fun delete(
        account: Account,
        id: Long,
    ): Boolean {
        val contact = contactRepository.findByIdAndAccountId(id, account.id!!) ?: return false
        contactRepository.delete(contact)
        return true
    }

    private fun mergeUpdate(
        existing: Contact,
        incoming: Contact,
    ): Contact {
        incoming.firstName?.let { existing.firstName = it }
        incoming.lastName?.let { existing.lastName = it }
        incoming.timeZone?.let { existing.timeZone = it }
        if (incoming.customFields.isNotEmpty()) {
            existing.customFields = incoming.customFields
        }
        return existing
    }
}
