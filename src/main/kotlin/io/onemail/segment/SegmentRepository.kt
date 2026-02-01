package io.onemail.segment

import org.springframework.data.jpa.repository.JpaRepository
import org.springframework.data.jpa.repository.JpaSpecificationExecutor

interface SegmentRepository :
    JpaRepository<Segment, Long>,
    JpaSpecificationExecutor<Segment> {
    fun findByIdAndAccountId(
        id: Long,
        accountId: Long,
    ): Segment?
}
