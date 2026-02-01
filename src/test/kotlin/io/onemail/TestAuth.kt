package io.onemail

import io.onemail.account.Account
import org.springframework.security.authentication.UsernamePasswordAuthenticationToken
import org.springframework.security.core.context.SecurityContextHolder
import org.springframework.security.oauth2.core.DefaultOAuth2AuthenticatedPrincipal

object TestAuth {
    fun withAccount(account: Account) {
        val principal =
            DefaultOAuth2AuthenticatedPrincipal(
                account.id.toString(),
                mapOf(
                    "account" to account,
                    "tokenName" to "test-token",
                ),
                emptyList(),
            )
        val authentication = UsernamePasswordAuthenticationToken(principal, "", emptyList())
        SecurityContextHolder.getContext().authentication = authentication
    }
}
