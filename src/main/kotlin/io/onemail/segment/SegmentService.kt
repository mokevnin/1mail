package io.onemail.segment

import io.onemail.account.Account
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.stereotype.Service
import org.springframework.transaction.annotation.Transactional

@Service
class SegmentService(
    private val segmentRepository: SegmentRepository,
) {
    @Transactional
    fun create(
        account: Account,
        segment: Segment,
    ): Segment {
        if (segment.account.id == null) {
            segment.account = account
        }
        return segmentRepository.save(segment)
    }

    fun list(
        account: Account,
        pageable: Pageable,
    ): Page<Segment> {
        val spec = SegmentSpecifications.hasAccountId(account.id!!)
        return segmentRepository.findAll(spec, pageable)
    }

    fun get(
        account: Account,
        id: Long,
    ): Segment? = segmentRepository.findByIdAndAccountId(id, account.id!!)

    @Transactional
    fun update(
        account: Account,
        id: Long,
        segment: Segment,
    ): Segment? {
        val existing = segmentRepository.findByIdAndAccountId(id, account.id!!) ?: return null
        val updated = mergeUpdate(existing, segment)
        return segmentRepository.save(updated)
    }

    @Transactional
    fun delete(
        account: Account,
        id: Long,
    ): Boolean {
        val segment = segmentRepository.findByIdAndAccountId(id, account.id!!) ?: return false
        segmentRepository.delete(segment)
        return true
    }

    private fun mergeUpdate(
        existing: Segment,
        incoming: Segment,
    ): Segment {
        incoming.name.takeIf { it.isNotBlank() }?.let { existing.name = it }
        existing.type = incoming.type
        existing.definition = incoming.definition
        return existing
    }
}
