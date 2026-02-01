package io.onemail.token

import io.onemail.account.Account
import jakarta.persistence.*
import org.springframework.data.annotation.CreatedDate
import org.springframework.data.jpa.domain.support.AuditingEntityListener
import java.time.Instant

@Entity
@EntityListeners(AuditingEntityListener::class)
class ApiToken(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    var id: Long? = null,
    @ManyToOne(fetch = FetchType.LAZY)
    var account: Account = Account(),
    @Column(unique = true)
    var token: String = "",
    var name: String? = null,
    @CreatedDate
    var createdAt: Instant? = null,
)
