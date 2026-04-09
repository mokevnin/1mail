package io.onemail.contact

import io.onemail.model.ContactStatus
import jakarta.persistence.CollectionTable
import jakarta.persistence.Column
import jakarta.persistence.ElementCollection
import jakarta.persistence.Entity
import jakarta.persistence.EntityListeners
import jakarta.persistence.EnumType
import jakarta.persistence.Enumerated
import jakarta.persistence.GeneratedValue
import jakarta.persistence.GenerationType
import jakarta.persistence.Id
import jakarta.persistence.JoinColumn
import jakarta.persistence.MapKeyColumn
import jakarta.persistence.Table
import org.springframework.data.annotation.CreatedDate
import org.springframework.data.annotation.LastModifiedDate
import org.springframework.data.jpa.domain.support.AuditingEntityListener
import java.time.Instant

@Entity
@Table(name = "contacts")
@EntityListeners(AuditingEntityListener::class)
class Contact(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    var id: Long? = null,
    @Column(nullable = false, unique = true)
    var email: String = "",
    @Enumerated(EnumType.STRING)
    @Column(nullable = false)
    var status: ContactStatus = ContactStatus.active,
    @CreatedDate
    @Column(nullable = false, updatable = false)
    var createdAt: Instant = Instant.EPOCH,
    @LastModifiedDate
    @Column(nullable = false)
    var updatedAt: Instant = Instant.EPOCH,
    @Column
    var firstName: String? = null,
    @Column
    var lastName: String? = null,
    @Column
    var timeZone: String? = null,
    @ElementCollection
    @CollectionTable(name = "contact_custom_fields", joinColumns = [JoinColumn(name = "contact_id")])
    @MapKeyColumn(name = "field_key")
    @Column(name = "field_value")
    var customFields: MutableMap<String, String> = mutableMapOf(),
)
