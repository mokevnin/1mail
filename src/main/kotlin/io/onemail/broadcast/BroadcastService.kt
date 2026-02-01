package io.onemail.broadcast

import io.onemail.account.Account
import io.onemail.segment.SegmentRepository
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.http.HttpStatus
import org.springframework.stereotype.Service
import org.springframework.transaction.annotation.Transactional
import org.springframework.web.server.ResponseStatusException

@Service
class BroadcastService(
    private val broadcastRepository: BroadcastRepository,
    private val segmentRepository: SegmentRepository,
) {
    @Transactional
    fun create(
        account: Account,
        broadcast: Broadcast,
        segmentId: Long,
    ): Broadcast {
        if (broadcast.account.id == null) {
            broadcast.account = account
        }
        val segment =
            segmentRepository.findByIdAndAccountId(segmentId, account.id!!)
                ?: throw ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "Segment not found")
        broadcast.segment = segment
        return broadcastRepository.save(broadcast)
    }

    fun list(
        account: Account,
        pageable: Pageable,
    ): Page<Broadcast> {
        val spec = BroadcastSpecifications.hasAccountId(account.id!!)
        return broadcastRepository.findAll(spec, pageable)
    }

    fun get(
        account: Account,
        id: Long,
    ): Broadcast? = broadcastRepository.findByIdAndAccountId(id, account.id!!)

    @Transactional
    fun update(
        account: Account,
        id: Long,
        broadcast: Broadcast,
        segmentId: Long?,
    ): Broadcast? {
        val existing = broadcastRepository.findByIdAndAccountId(id, account.id!!) ?: return null
        val updated = mergeUpdate(existing, broadcast)
        segmentId?.let {
            val segment =
                segmentRepository.findByIdAndAccountId(it, account.id!!)
                    ?: throw ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "Segment not found")
            updated.segment = segment
        }
        return broadcastRepository.save(updated)
    }

    @Transactional
    fun delete(
        account: Account,
        id: Long,
    ): Boolean {
        val broadcast = broadcastRepository.findByIdAndAccountId(id, account.id!!) ?: return false
        broadcastRepository.delete(broadcast)
        return true
    }

    private fun mergeUpdate(
        existing: Broadcast,
        incoming: Broadcast,
    ): Broadcast {
        incoming.name.takeIf { it.isNotBlank() }?.let { existing.name = it }
        existing.status = incoming.status
        return existing
    }
}
