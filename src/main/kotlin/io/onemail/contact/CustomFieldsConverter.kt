package io.onemail.contact

import com.fasterxml.jackson.databind.ObjectMapper
import jakarta.persistence.AttributeConverter
import jakarta.persistence.Converter

@Converter
class CustomFieldsConverter : AttributeConverter<Map<String, String>, String> {
    private val mapper = ObjectMapper()

    override fun convertToDatabaseColumn(attribute: Map<String, String>?): String =
        mapper.writeValueAsString(attribute ?: emptyMap<String, String>())

    override fun convertToEntityAttribute(dbData: String?): Map<String, String> {
        if (dbData.isNullOrBlank()) return emptyMap()
        return mapper.readValue(dbData, Map::class.java) as Map<String, String>
    }
}
