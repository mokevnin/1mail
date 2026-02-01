package io.onemail.broadcast

import io.onemail.account.Account
import io.onemail.segment.Segment
import jakarta.persistence.*
import org.springframework.data.annotation.CreatedDate
import org.springframework.data.annotation.LastModifiedDate
import org.springframework.data.jpa.domain.support.AuditingEntityListener
import java.time.Instant

@Entity
@EntityListeners(AuditingEntityListener::class)
class Broadcast(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    var id: Long? = null,
    @ManyToOne(fetch = FetchType.LAZY)
    var account: Account = Account(),
    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "segment_id")
    var segment: Segment? = null,
    var name: String = "",
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    var status: BroadcastStatus = BroadcastStatus.DRAFT,
    @CreatedDate
    var createdAt: Instant? = null,
    @LastModifiedDate
    var updatedAt: Instant? = null,
)

enum class BroadcastStatus {
    DRAFT,
    SCHEDULED,
    SENT,
}
