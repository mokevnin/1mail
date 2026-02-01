package io.onemail.segment

import io.onemail.account.Account
import io.onemail.model.CreateSegmentRequest
import io.onemail.model.UpdateSegmentRequest
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
import io.onemail.model.Segment as SegmentDto

@Mapper(componentModel = "spring", unmappedTargetPolicy = ReportingPolicy.ERROR)
interface SegmentMapper {
    fun toDto(entity: Segment): SegmentDto

    @ValueMappings(
        ValueMapping(source = "RULE", target = "rule"),
        ValueMapping(source = "SNAPSHOT", target = "snapshot"),
    )
    fun toDtoType(type: SegmentType): io.onemail.model.SegmentType

    @ValueMappings(
        ValueMapping(source = "rule", target = "RULE"),
        ValueMapping(source = "snapshot", target = "SNAPSHOT"),
    )
    fun toEntityType(type: io.onemail.model.SegmentType): SegmentType

    @BeanMapping(nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", expression = "java(account)")
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    fun fromCreate(
        request: CreateSegmentRequest,
        @Context account: Account,
    ): Segment

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "name", source = "name", qualifiedByName = ["unwrapNullable"], conditionQualifiedByName = ["isPresent"])
    @Mapping(target = "type", source = "type", qualifiedByName = ["unwrapNullableType"], conditionQualifiedByName = ["isPresent"])
    @Mapping(target = "definition", source = "definition", qualifiedByName = ["unwrapNullable"], conditionQualifiedByName = ["isPresent"])
    fun updateFromUpdate(
        request: UpdateSegmentRequest,
        @MappingTarget entity: Segment,
    )

    fun mapId(id: Long?): Long = requireNotNull(id) { "id not set" }

    fun mapInstant(instant: Instant?): String = requireNotNull(instant) { "timestamp not set" }.toString()

    @Named("unwrapNullable")
    fun unwrapNullable(value: JsonNullable<String>?): String = if (value?.isPresent == true) value.orElse("") else ""

    @Named("unwrapNullableType")
    fun unwrapNullableType(value: JsonNullable<io.onemail.model.SegmentType>?): SegmentType =
        if (value?.isPresent == true) {
            value.orElse(null)?.let { toEntityType(it) } ?: SegmentType.RULE
        } else {
            SegmentType.RULE
        }

    @Named("isPresent")
    fun isPresent(value: JsonNullable<*>?): Boolean = value?.isPresent == true
}
