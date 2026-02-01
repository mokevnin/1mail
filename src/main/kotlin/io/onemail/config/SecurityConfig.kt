package io.onemail.config

import io.onemail.token.ApiTokenRepository
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.security.config.annotation.web.builders.HttpSecurity
import org.springframework.security.config.annotation.web.configuration.EnableWebSecurity
import org.springframework.security.oauth2.core.DefaultOAuth2AuthenticatedPrincipal
import org.springframework.security.oauth2.core.OAuth2AuthenticatedPrincipal
import org.springframework.security.oauth2.server.resource.introspection.BadOpaqueTokenException
import org.springframework.security.oauth2.server.resource.introspection.OpaqueTokenIntrospector
import org.springframework.security.web.SecurityFilterChain

@Configuration
@EnableWebSecurity
class SecurityConfig {
    @Bean
    fun filterChain(http: HttpSecurity): SecurityFilterChain {
        http
            .authorizeHttpRequests { authorize ->
                authorize
                    .requestMatchers("/h2-console/**")
                    .permitAll()
                    .requestMatchers("/api/**")
                    .authenticated()
                    .anyRequest()
                    .permitAll()
            }.oauth2ResourceServer { oauth2 ->
                oauth2.opaqueToken { }
            }.csrf { it.disable() }
            .headers { headers -> headers.frameOptions { it.disable() } }
        return http.build()
    }

    @Bean
    fun introspector(apiTokenRepository: ApiTokenRepository): OpaqueTokenIntrospector =
        OpaqueTokenIntrospector { token ->
            val apiToken =
                apiTokenRepository.findByToken(token)
                    ?: throw BadOpaqueTokenException("Invalid API token")

            DefaultOAuth2AuthenticatedPrincipal(
                apiToken.account.id.toString(),
                mapOf<String, Any>(
                    "account" to apiToken.account,
                    "tokenName" to (apiToken.name ?: ""),
                ),
                emptyList(),
            )
        }
}
