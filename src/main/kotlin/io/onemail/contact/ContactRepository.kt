package io.onemail.contact

import org.springframework.data.jpa.repository.JpaRepository

interface ContactRepository : JpaRepository<Contact, Long>
