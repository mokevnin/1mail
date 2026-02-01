package io.onemail.segment

import io.onemail.account.Account
import jakarta.persistence.*
import org.springframework.data.annotation.CreatedDate
import org.springframework.data.annotation.LastModifiedDate
import org.springframework.data.jpa.domain.support.AuditingEntityListener
import java.time.Instant

@Entity
@EntityListeners(AuditingEntityListener::class)
class Segment(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    var id: Long? = null,
    @ManyToOne(fetch = FetchType.LAZY)
    var account: Account = Account(),
    var name: String = "",
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    var type: SegmentType = SegmentType.RULE,
    @Lob
    var definition: String? = null,
    @CreatedDate
    var createdAt: Instant? = null,
    @LastModifiedDate
    var updatedAt: Instant? = null,
)

enum class SegmentType {
    RULE,
    SNAPSHOT,
}
