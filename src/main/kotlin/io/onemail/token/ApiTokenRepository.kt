package io.onemail.token

import org.springframework.data.jpa.repository.JpaRepository

interface ApiTokenRepository : JpaRepository<ApiToken, Long> {
    fun findByToken(token: String): ApiToken?
}
