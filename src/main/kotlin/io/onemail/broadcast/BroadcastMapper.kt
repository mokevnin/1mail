package io.onemail.broadcast

import io.onemail.account.Account
import io.onemail.model.CreateBroadcastRequest
import io.onemail.model.UpdateBroadcastRequest
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
import io.onemail.model.Broadcast as BroadcastDto

@Mapper(componentModel = "spring", unmappedTargetPolicy = ReportingPolicy.ERROR)
interface BroadcastMapper {
    @Mapping(target = "segmentId", source = "segment")
    fun toDto(entity: Broadcast): BroadcastDto

    @ValueMappings(
        ValueMapping(source = "DRAFT", target = "draft"),
        ValueMapping(source = "SCHEDULED", target = "scheduled"),
        ValueMapping(source = "SENT", target = "sent"),
    )
    fun toDtoStatus(status: BroadcastStatus): io.onemail.model.BroadcastStatus

    @ValueMappings(
        ValueMapping(source = "draft", target = "DRAFT"),
        ValueMapping(source = "scheduled", target = "SCHEDULED"),
        ValueMapping(source = "sent", target = "SENT"),
    )
    fun toEntityStatus(status: io.onemail.model.BroadcastStatus): BroadcastStatus

    @BeanMapping(nullValuePropertyMappingStrategy = NullValuePropertyMappingStrategy.IGNORE)
    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", expression = "java(account)")
    @Mapping(target = "segment", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    fun fromCreate(
        request: CreateBroadcastRequest,
        @Context account: Account,
    ): Broadcast

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "account", ignore = true)
    @Mapping(target = "segment", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "name", source = "name", qualifiedByName = ["unwrapNullable"], conditionQualifiedByName = ["isPresent"])
    @Mapping(target = "status", source = "status", qualifiedByName = ["unwrapNullableStatus"], conditionQualifiedByName = ["isPresent"])
    fun updateFromUpdate(
        request: UpdateBroadcastRequest,
        @MappingTarget entity: Broadcast,
    )

    fun mapId(id: Long?): Long = requireNotNull(id) { "id not set" }

    fun mapInstant(instant: Instant?): String = requireNotNull(instant) { "timestamp not set" }.toString()

    fun mapSegmentId(segment: io.onemail.segment.Segment?): Long? = segment?.id

    @Named("unwrapNullable")
    fun unwrapNullable(value: JsonNullable<String>?): String = if (value?.isPresent == true) value.orElse("") else ""

    @Named("unwrapNullableStatus")
    fun unwrapNullableStatus(value: JsonNullable<io.onemail.model.BroadcastStatus>?): BroadcastStatus =
        if (value?.isPresent == true) {
            value.orElse(null)?.let { toEntityStatus(it) } ?: BroadcastStatus.DRAFT
        } else {
            BroadcastStatus.DRAFT
        }

    @Named("isPresent")
    fun isPresent(value: JsonNullable<*>?): Boolean = value?.isPresent == true
}
