package io.onemail.contact

import io.onemail.account.Account
import jakarta.persistence.*
import org.springframework.data.annotation.CreatedDate
import org.springframework.data.annotation.LastModifiedDate
import org.springframework.data.jpa.domain.support.AuditingEntityListener
import java.time.Instant

@Entity
@Table(uniqueConstraints = [UniqueConstraint(columnNames = ["account_id", "email"])])
@EntityListeners(AuditingEntityListener::class)
class Contact(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    var id: Long? = null,
    @ManyToOne(fetch = FetchType.LAZY)
    var account: Account = Account(),
    var email: String = "",
    var firstName: String? = null,
    var lastName: String? = null,
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    var status: ContactStatus = ContactStatus.ACTIVE,
    var timeZone: String? = null,
    @Convert(converter = CustomFieldsConverter::class)
    var customFields: Map<String, String> = emptyMap(),
    @CreatedDate
    var createdAt: Instant? = null,
    @LastModifiedDate
    var updatedAt: Instant? = null,
)

enum class ContactStatus {
    ACTIVE,
    UNSUBSCRIBED,
}
