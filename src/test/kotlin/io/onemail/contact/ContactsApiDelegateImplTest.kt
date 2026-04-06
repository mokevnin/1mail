package io.onemail.contact

import io.onemail.model.ContactStatus
import io.onemail.model.CreateContactRequest
import io.onemail.model.UpdateContactRequest
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import org.instancio.junit.Given
import org.instancio.junit.InstancioExtension
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.data.domain.PageRequest
import org.springframework.data.domain.Sort
import org.springframework.http.HttpStatus
import org.springframework.test.context.ActiveProfiles
import org.springframework.web.server.ResponseStatusException

@SpringBootTest
@ActiveProfiles("test")
@ExtendWith(InstancioExtension::class)
class ContactsApiDelegateImplTest {

    @Autowired
    private lateinit var contactService: ContactService

    @Test
    fun `contacts crud flow works`(@Given createContact: Contact, @Given updateContact: Contact) {
        val createRequest = CreateContactRequest(
            email = createContact.email,
            firstName = createContact.firstName,
            lastName = createContact.lastName,
            timeZone = createContact.timeZone,
            customFields = createContact.customFields,
        )

        val created = contactService.create(createRequest)

        val fetched = contactService.get(created.id)
        assertEquals(createRequest.email, fetched.email)
        assertEquals(ContactStatus.active, fetched.status)

        val list = contactService.list(
            pageable = PageRequest.of(0, 10, Sort.by(Sort.Order.desc("email"))),
            status = ContactStatus.active,
        )
        assertEquals(1, list.totalElements)
        assertEquals(createRequest.email, list.content.single().email)

        val updateRequest = UpdateContactRequest(
            firstName = updateContact.firstName,
            lastName = updateContact.lastName,
            timeZone = updateContact.timeZone,
            customFields = updateContact.customFields,
        )

        val updated = contactService.update(
            id = created.id,
            request = updateRequest,
        )
        assertEquals(updateRequest.firstName, updated.firstName)
        assertEquals(updateRequest.timeZone, updated.timeZone)
        assertEquals(updateRequest.customFields, updated.customFields)

        contactService.delete(created.id)

        val exception = assertFailsWith<ResponseStatusException> {
            contactService.get(created.id)
        }
        assertEquals(HttpStatus.NOT_FOUND, exception.statusCode)
    }
}
