package io.onemail

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import io.onemail.model.CreateContactRequest
import org.instancio.junit.Given
import org.instancio.junit.InstancioExtension
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.extension.ExtendWith
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
@ExtendWith(InstancioExtension::class)
class ApiExceptionHandlerTest {
    private val objectMapper = jacksonObjectMapper()

    @Autowired
    private lateinit var mockMvc: MockMvc

    @Test
    fun `duplicate contact email returns conflict problem details`(@Given request: CreateContactRequest) {
        val requestBody = objectMapper.writeValueAsString(request)

        mockMvc.perform(
            post("/api/contacts")
                .contentType(MediaType.APPLICATION_JSON)
                .content(requestBody),
        ).andExpect(status().isCreated)

        mockMvc.perform(
            post("/api/contacts")
                .contentType(MediaType.APPLICATION_JSON)
                .content(requestBody),
        )
            .andExpect(status().isConflict)
    }
}
