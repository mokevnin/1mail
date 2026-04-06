package io.onemail.contact

import io.onemail.model.ContactStatus
import org.springframework.data.domain.Page
import org.springframework.data.domain.Pageable
import org.springframework.data.jpa.repository.JpaRepository

interface ContactRepository : JpaRepository<Contact, Long> {
    fun findAllByStatus(status: ContactStatus, pageable: Pageable): Page<Contact>
}
