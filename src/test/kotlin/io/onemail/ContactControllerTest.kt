package io.onemail

import io.onemail.account.Account
import io.onemail.account.AccountRepository
import io.onemail.contact.ContactRepository
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.http.MediaType
import org.springframework.test.context.ActiveProfiles
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.delete
import org.springframework.test.web.servlet.get
import org.springframework.test.web.servlet.post
import org.springframework.test.web.servlet.put
import org.springframework.test.web.servlet.setup.MockMvcBuilders
import org.springframework.transaction.annotation.Transactional
import org.springframework.web.context.WebApplicationContext

@SpringBootTest
@ActiveProfiles("test")
@Transactional
class ContactControllerTest {
    private lateinit var mockMvc: MockMvc

    private val objectMapper = TestObjectMapperProvider.objectMapper

    @Autowired
    private lateinit var webApplicationContext: WebApplicationContext

    @Autowired
    private lateinit var accountRepository: AccountRepository

    @Autowired
    private lateinit var contactRepository: ContactRepository

    private lateinit var account: Account

    @BeforeEach
    fun setUp() {
        account = accountRepository.save(TestDataFactory.account())
        TestAuth.withAccount(account)
        mockMvc = MockMvcBuilders.webAppContextSetup(webApplicationContext).build()
    }

    @Test
    fun `create contact returns 201`() {
        val body = objectMapper.writeValueAsString(TestDataFactory.createContactRequest())

        mockMvc
            .post("/api/contacts") {
                contentType = MediaType.APPLICATION_JSON
                content = body
            }.andExpect {
                status { isCreated() }
            }
    }

    @Test
    fun `list contacts returns 200`() {
        contactRepository.save(TestDataFactory.contact(account))

        mockMvc
            .get("/api/contacts")
            .andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `get contact returns 200`() {
        val contact = contactRepository.save(TestDataFactory.contact(account))

        mockMvc
            .get("/api/contacts/${contact.id}")
            .andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `update contact returns 200`() {
        val contact = contactRepository.save(TestDataFactory.contact(account))

        mockMvc
            .put("/api/contacts/${contact.id}") {
                contentType = MediaType.APPLICATION_JSON
                content = "{}"
            }.andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `delete contact returns 204`() {
        val contact = contactRepository.save(TestDataFactory.contact(account))

        mockMvc
            .delete("/api/contacts/${contact.id}")
            .andExpect {
                status { isNoContent() }
            }
    }
}
