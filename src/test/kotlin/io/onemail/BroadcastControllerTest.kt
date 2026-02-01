package io.onemail

import io.onemail.account.Account
import io.onemail.account.AccountRepository
import io.onemail.broadcast.BroadcastRepository
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
class BroadcastControllerTest {
    private lateinit var mockMvc: MockMvc

    private val objectMapper = TestObjectMapperProvider.objectMapper

    @Autowired
    private lateinit var webApplicationContext: WebApplicationContext

    @Autowired
    private lateinit var accountRepository: AccountRepository

    @Autowired
    private lateinit var segmentRepository: SegmentRepository

    @Autowired
    private lateinit var broadcastRepository: BroadcastRepository

    private lateinit var account: Account

    @BeforeEach
    fun setUp() {
        account = accountRepository.save(TestDataFactory.account())
        TestAuth.withAccount(account)
        mockMvc = MockMvcBuilders.webAppContextSetup(webApplicationContext).build()
    }

    @Test
    fun `create broadcast returns 201`() {
        val segment = segmentRepository.save(TestDataFactory.segment(account, SegmentType.SNAPSHOT))

        val body = objectMapper.writeValueAsString(TestDataFactory.createBroadcastRequest(segment.id!!))

        mockMvc
            .post("/api/broadcasts") {
                contentType = MediaType.APPLICATION_JSON
                content = body
            }.andExpect {
                status { isCreated() }
            }
    }

    @Test
    fun `list broadcasts returns 200`() {
        val segment = segmentRepository.save(TestDataFactory.segment(account, SegmentType.SNAPSHOT))
        broadcastRepository.save(TestDataFactory.broadcast(account, segment))

        mockMvc
            .get("/api/broadcasts")
            .andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `get broadcast returns 200`() {
        val segment = segmentRepository.save(TestDataFactory.segment(account, SegmentType.SNAPSHOT))
        val broadcast = broadcastRepository.save(TestDataFactory.broadcast(account, segment))

        mockMvc
            .get("/api/broadcasts/${broadcast.id}")
            .andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `update broadcast returns 200`() {
        val segment = segmentRepository.save(TestDataFactory.segment(account, SegmentType.SNAPSHOT))
        val broadcast = broadcastRepository.save(TestDataFactory.broadcast(account, segment))

        mockMvc
            .put("/api/broadcasts/${broadcast.id}") {
                contentType = MediaType.APPLICATION_JSON
                content = "{}"
            }.andExpect {
                status { isOk() }
            }
    }

    @Test
    fun `delete broadcast returns 204`() {
        val segment = segmentRepository.save(TestDataFactory.segment(account, SegmentType.SNAPSHOT))
        val broadcast = broadcastRepository.save(TestDataFactory.broadcast(account, segment))

        mockMvc
            .delete("/api/broadcasts/${broadcast.id}")
            .andExpect {
                status { isNoContent() }
            }
    }
}
