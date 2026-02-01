package io.onemail.contact

import io.onemail.account.Account
import io.onemail.model.CreateContactRequest
import io.onemail.model.UpdateContactRequest
import org.mapstruct.BeanMapping
import org.mapstruct.Context
import org.mapstruct.Mapper
import org.mapstruct.Mapping
import org.mapstruct.MappingTarget
import org.mapstruct.Named
import org.mapstruct.NullValuePropertyMappingStrategy
import org.mapstruct.ReportingPolicy
import org.mapstruct.ValueMapping
import org.mapstruct.ValueMappings
import org.openapitools.jackson.nullable.JsonNullable
import java.time.Instant
import io.onemail.model.Contact as ContactDto

@Mapper(componentModel = "spring", unmappedTargetPolicy = ReportingPolicy.ERROR)
interface ContactMapper {
    fun toDto(entity: Contact): ContactDto

    @ValueMappings(
        ValueMapping(source = "ACTIVE", target = "active"),
        ValueMapping(source = "UNSUBSCRIBED", target = "unsubscribed"),
    )
    fun toDtoStatus(status: ContactStatus): io.onemail.model.ContactStatus

    @ValueMappings(
        ValueMapping(source = "active", target = "ACTIVE"),
        ValueMapping(source = "unsubscribed", target = "UNSUBSCRIBED"),
    )
    fun toEntityStatus(status: io.onemail.model.ContactStatus): ContactStatus

    @BeanMapping(nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", expression = "java(account)")
    @Mapping(target = "status", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    fun fromCreate(
        request: CreateContactRequest,
        @Context account: Account,
    ): Contact

    @BeanMapping(nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", ignore = true)
    @Mapping(target = "email", ignore = true)
    @Mapping(target = "status", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    fun updateFromCreate(
        request: CreateContactRequest,
        @MappingTarget entity: Contact,
    )

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", ignore = true)
    @Mapping(target = "email", ignore = true)
    @Mapping(target = "status", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "firstName", source = "firstName", qualifiedByName = ["unwrapNullable"], conditionQualifiedByName = ["isPresent"])
    @Mapping(target = "lastName", source = "lastName", qualifiedByName = ["unwrapNullable"], conditionQualifiedByName = ["isPresent"])
    @Mapping(target = "timeZone", source = "timeZone", qualifiedByName = ["unwrapNullable"], conditionQualifiedByName = ["isPresent"])
    @Mapping(
        target = "customFields",
        source = "customFields",
        qualifiedByName = ["unwrapNullableMap"],
        conditionQualifiedByName = ["isPresentMap"],
    )
    fun updateFromUpdate(
        request: UpdateContactRequest,
        @MappingTarget entity: Contact,
    )

    fun mapId(id: Long?): Long = requireNotNull(id) { "id not set" }

    fun mapInstant(instant: Instant?): String = requireNotNull(instant) { "timestamp not set" }.toString()

    fun mapCustomFields(fields: Map<String, String>): Map<String, String>? = fields.takeIf { it.isNotEmpty() }

    @Named("unwrapNullable")
    fun unwrapNullable(value: JsonNullable<String>?): String? = if (value?.isPresent == true) value.orElse(null) else null

    @Named("unwrapNullableMap")
    fun unwrapNullableMap(value: JsonNullable<Map<String, String>>?): Map<String, String> =
        if (value?.isPresent == true) value.orElse(emptyMap()) else emptyMap()

    @Named("isPresent")
    fun isPresent(value: JsonNullable<*>?): Boolean = value?.isPresent == true

    @Named("isPresentMap")
    fun isPresentMap(value: JsonNullable<Map<String, String>>?): Boolean = value?.isPresent == true
}
