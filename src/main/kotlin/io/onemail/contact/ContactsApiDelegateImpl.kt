package io.onemail.contact

import io.onemail.api.ContactsApiDelegate
import io.onemail.model.ContactPage
import io.onemail.model.ContactResource
import io.onemail.model.ContactStatus
import io.onemail.model.CreateContactInput
import io.onemail.model.UpdateContactInput
import org.springframework.data.domain.Pageable
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.stereotype.Service

@Service
class ContactsApiDelegateImpl(
    private val contactService: ContactService,
) : ContactsApiDelegate {

    override fun contactsCreate(createContactInput: CreateContactInput): ResponseEntity<ContactResource> {
        return ResponseEntity.status(HttpStatus.CREATED).body(contactService.create(createContactInput))
    }

    override fun contactsDelete(id: Long): ResponseEntity<Unit> {
        contactService.delete(id)
        return ResponseEntity.noContent().build()
    }

    override fun contactsGet(id: Long): ResponseEntity<ContactResource> {
        return ResponseEntity.ok(contactService.get(id))
    }

    override fun contactsList(
        status: ContactStatus?,
        pageable: Pageable,
    ): ResponseEntity<ContactPage> {
        return ResponseEntity.ok(contactService.list(pageable, status))
    }

    override fun contactsUpdate(id: Long, updateContactInput: UpdateContactInput): ResponseEntity<ContactResource> {
        return ResponseEntity.ok(contactService.update(id, updateContactInput))
    }
}
