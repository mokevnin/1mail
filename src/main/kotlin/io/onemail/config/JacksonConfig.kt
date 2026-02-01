package io.onemail.config

import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import org.openapitools.jackson.nullable.JsonNullableModule
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.context.annotation.Primary

@Configuration
class JacksonConfig {
    @Bean
    fun jsonNullableModule(): JsonNullableModule = JsonNullableModule()

    @Bean
    @Primary
    fun objectMapper(): ObjectMapper = jacksonObjectMapper().registerModule(JsonNullableModule())
}
