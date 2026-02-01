package io.onemail.contact

import io.onemail.account.Account
import io.onemail.model.CreateContactRequest
import io.onemail.model.UpdateContactRequest
import org.springframework.data.domain.Pageable
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.security.core.context.SecurityContextHolder
import org.springframework.security.oauth2.core.OAuth2AuthenticatedPrincipal
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.PutMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ResponseStatusException
import io.onemail.model.Contact as ContactDto
import io.onemail.model.ContactStatus as ContactStatusDto

@RestController
class ContactController(
    private val contactService: ContactService,
    private val contactMapper: ContactMapper,
) {
    @PostMapping("/api/contacts")
    fun contactsCreate(
        @jakarta.validation.Valid @RequestBody createContactRequest: CreateContactRequest,
    ): ResponseEntity<ContactDto> {
        val contact = contactMapper.fromCreate(createContactRequest, account())
        val saved = contactService.create(account(), contact)
        return ResponseEntity
            .status(HttpStatus.CREATED)
            .body(contactMapper.toDto(saved))
    }

    @GetMapping("/api/contacts")
    fun contactsList(
        pageable: Pageable,
        @RequestParam status: ContactStatusDto?,
    ): ResponseEntity<org.springframework.data.domain.Page<ContactDto>> =
        ResponseEntity.ok(
            contactService
                .list(account(), pageable, status?.let { contactMapper.toEntityStatus(it) })
                .map(contactMapper::toDto),
        )

    @GetMapping("/api/contacts/{id}")
    fun contactsGet(
        @PathVariable id: String,
    ): ResponseEntity<ContactDto> {
        val contact =
            contactService.get(account(), parseId(id))
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Contact not found")
        return ResponseEntity.ok(contactMapper.toDto(contact))
    }

    @PutMapping("/api/contacts/{id}")
    fun contactsUpdate(
        @PathVariable id: String,
        @jakarta.validation.Valid @RequestBody updateContactRequest: UpdateContactRequest,
    ): ResponseEntity<ContactDto> {
        val existing =
            contactService.get(account(), parseId(id))
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Contact not found")
        contactMapper.updateFromUpdate(updateContactRequest, existing)
        val saved =
            contactService.update(account(), parseId(id), existing)
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Contact not found")
        return ResponseEntity.ok(contactMapper.toDto(saved))
    }

    @DeleteMapping("/api/contacts/{id}")
    fun contactsDelete(
        @PathVariable id: String,
    ): ResponseEntity<Void> {
        if (!contactService.delete(account(), parseId(id))) {
            throw ResponseStatusException(HttpStatus.NOT_FOUND, "Contact not found")
        }
        return ResponseEntity.noContent().build()
    }

    private fun account(): Account {
        val principal = SecurityContextHolder.getContext().authentication!!.principal as OAuth2AuthenticatedPrincipal
        return principal.getAttribute("account")!!
    }

    private fun parseId(id: String): Long =
        try {
            id.toLong()
        } catch (e: IllegalArgumentException) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "Invalid ID format")
        }
}
