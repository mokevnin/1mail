package io.onemail.contact

import io.onemail.api.ContactsApiDelegate
import io.onemail.model.ContactPageResponse
import io.onemail.model.ContactResponse
import io.onemail.model.ContactStatus
import io.onemail.model.CreateContactRequest
import io.onemail.model.UpdateContactRequest
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.stereotype.Service
import org.springframework.data.domain.Pageable

@Service
class ContactsApiDelegateImpl(
    private val contactService: ContactService,
) : ContactsApiDelegate {

    override fun contactsCreate(createContactRequest: CreateContactRequest): ResponseEntity<ContactResponse> {
        return ResponseEntity.status(HttpStatus.CREATED).body(contactService.create(createContactRequest))
    }

    override fun contactsDelete(id: Long): ResponseEntity<Unit> {
        contactService.delete(id)
        return ResponseEntity.noContent().build()
    }

    override fun contactsGet(id: Long): ResponseEntity<ContactResponse> {
        return ResponseEntity.ok(contactService.get(id))
    }

    override fun contactsList(
        status: ContactStatus?,
        pageable: Pageable,
    ): ResponseEntity<ContactPageResponse> {
        return ResponseEntity.ok(contactService.list(pageable, status))
    }

    override fun contactsUpdate(id: Long, updateContactRequest: UpdateContactRequest): ResponseEntity<ContactResponse> {
        return ResponseEntity.ok(contactService.update(id, updateContactRequest))
    }
}
