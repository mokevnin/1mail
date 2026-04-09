package io.onemail

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import io.onemail.model.CreateContactInput
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc
import org.springframework.http.MediaType
import org.springframework.test.context.ActiveProfiles
import org.springframework.test.web.servlet.MockMvc
import org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post
import org.springframework.test.web.servlet.result.MockMvcResultMatchers.status

@SpringBootTest
@AutoConfigureMockMvc
@ActiveProfiles("test")
class ApiExceptionHandlerTest {
    private val objectMapper = jacksonObjectMapper()

    @Autowired
    private lateinit var mockMvc: MockMvc

    @Test
    fun `duplicate contact email returns conflict problem details`() {
        val request = CreateContactInput(
            email = "duplicate@example.com",
            firstName = "Alice",
            lastName = "Johnson",
            timeZone = "America/New_York",
            customFields = mapOf("source" to "test"),
        )
        val requestBody = objectMapper.writeValueAsString(request)

        mockMvc.perform(
            post("/contacts")
                .contentType(MediaType.APPLICATION_JSON)
                .content(requestBody),
        ).andExpect(status().isCreated)

        mockMvc.perform(
            post("/contacts")
                .contentType(MediaType.APPLICATION_JSON)
                .content(requestBody),
        )
            .andExpect(status().isConflict)
    }
}
