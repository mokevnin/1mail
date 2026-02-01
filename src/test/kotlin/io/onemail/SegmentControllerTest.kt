package io.onemail

import io.onemail.account.Account
import io.onemail.account.AccountRepository
import io.onemail.segment.SegmentRepository
import io.onemail.segment.SegmentType
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
class SegmentControllerTest {
    private lateinit var mockMvc: MockMvc

    private val objectMapper = TestObjectMapperProvider.objectMapper

    @Autowired
    private lateinit var webApplicationContext: WebApplicationContext

    @Autowired
    private lateinit var accountRepository: AccountRepository

    @Autowired
    private lateinit var segmentRepository: SegmentRepository

    private lateinit var account: Account

    @BeforeEach
    fun setUp() {
        account = accountRepository.save(TestDataFactory.account())
        TestAuth.withAccount(account)
        mockMvc = MockMvcBuilders.webAppContextSetup(webApplicationContext).build()
    }

    @Test
    fun `create segment returns 201`() {
        val body = objectMapper.writeValueAsString(TestDataFactory.createSegmentRequest(io.onemail.model.SegmentType.rule))

        mockMvc
            .post("/api/segments") {
                contentType = MediaType.APPLICATION_JSON
                content = body
            }.andExpect {
                status { isCreated() }
            }
    }

    @Test
    fun `list segments returns 200`() {
        segmentRepository.save(TestDataFactory.segment(account, SegmentType.RULE))

        mockMvc
            .get("/api/segments")
            .andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `get segment returns 200`() {
        val segment =
            TestDataFactory
                .segment(account, SegmentType.SNAPSHOT)
                .also {
                    it.type = SegmentType.SNAPSHOT
                    it.definition = null
                }.let { segmentRepository.save(it) }

        mockMvc
            .get("/api/segments/${segment.id}")
            .andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `delete segment returns 204`() {
        val segment = segmentRepository.save(TestDataFactory.segment(account, SegmentType.SNAPSHOT))

        mockMvc
            .delete("/api/segments/${segment.id}")
            .andExpect {
                status { isNoContent() }
            }
    }
}
