package io.onemail.broadcast

import org.springframework.data.jpa.repository.JpaRepository
import org.springframework.data.jpa.repository.JpaSpecificationExecutor

interface BroadcastRepository :
    JpaRepository<Broadcast, Long>,
    JpaSpecificationExecutor<Broadcast> {
    fun findByIdAndAccountId(
        id: Long,
        accountId: Long,
    ): Broadcast?
}
