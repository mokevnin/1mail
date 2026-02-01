package io.onemail

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import org.openapitools.jackson.nullable.JsonNullableModule

object TestObjectMapperProvider {
    val objectMapper = jacksonObjectMapper().registerModule(JsonNullableModule())
}
