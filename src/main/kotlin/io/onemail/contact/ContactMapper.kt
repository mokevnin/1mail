package io.onemail.contact

import io.onemail.model.ContactPageResponse
import io.onemail.model.ContactResponse
import io.onemail.model.CreateContactRequest
import io.onemail.model.Pageable
import io.onemail.model.Sort
import io.onemail.model.UpdateContactRequest
import org.mapstruct.Mapper
import org.mapstruct.Mapping
import org.mapstruct.MappingTarget
import org.mapstruct.Named
import org.springframework.data.domain.Page
import java.time.Instant

@Mapper(componentModel = "spring")
interface ContactMapper {

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "status", expression = "java(io.onemail.model.ContactStatus.active)")
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "customFields", source = "customFields", qualifiedByName = ["toMutableMap"])
    fun toEntity(request: CreateContactRequest): Contact

    @Mapping(target = "id", ignore = true)
    @Mapping(target = "email", ignore = true)
    @Mapping(target = "status", ignore = true)
    @Mapping(target = "createdAt", ignore = true)
    @Mapping(target = "updatedAt", ignore = true)
    @Mapping(target = "customFields", source = "customFields", qualifiedByName = ["toMutableMap"])
    fun updateEntity(request: UpdateContactRequest, @MappingTarget contact: Contact)

    @Mapping(target = "id", expression = "java(java.util.Objects.requireNonNull(contact.getId()))")
    @Mapping(target = "createdAt", source = "createdAt", qualifiedByName = ["instantToString"])
    @Mapping(target = "updatedAt", source = "updatedAt", qualifiedByName = ["instantToString"])
    @Mapping(target = "customFields", source = "customFields", qualifiedByName = ["emptyMapToNull"])
    fun toResponse(contact: Contact): ContactResponse

    fun toPageResponse(page: Page<Contact>): ContactPageResponse {
        return ContactPageResponse(
            content = page.content.map(::toResponse),
            totalElements = page.totalElements,
            totalPages = page.totalPages,
            propertySize = page.size,
            number = page.number,
            numberOfElements = page.numberOfElements,
            first = page.isFirst,
            last = page.isLast,
            empty = page.isEmpty,
            pageable = toPageable(page.pageable),
            sort = toSort(page.sort),
        )
    }

    fun toSort(sort: org.springframework.data.domain.Sort): Sort {
        return Sort(
            sorted = sort.isSorted,
            unsorted = sort.isUnsorted,
            empty = sort.isEmpty,
        )
    }

    fun toPageable(pageable: org.springframework.data.domain.Pageable): Pageable {
        return Pageable(
            pageSize = pageable.pageSize,
            pageNumber = pageable.pageNumber,
            offset = pageable.offset,
            paged = pageable.isPaged,
            unpaged = pageable.isUnpaged,
            sort = toSort(pageable.sort),
        )
    }

    @Named("instantToString")
    fun instantToString(value: Instant): String {
        return value.toString()
    }

    @Named("emptyMapToNull")
    fun emptyMapToNull(value: Map<String, String>?): Map<String, String>? {
        return value?.takeUnless { it.isEmpty() }
    }

    @Named("toMutableMap")
    fun toMutableMap(value: Map<String, String>?): MutableMap<String, String> {
        return value?.toMutableMap() ?: mutableMapOf()
    }
}
