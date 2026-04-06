package io.onemail.contact

import io.onemail.model.ContactPageResponse
import io.onemail.model.ContactResponse
import io.onemail.model.ContactStatus
import io.onemail.model.CreateContactRequest
import io.onemail.model.UpdateContactRequest
import org.springframework.data.domain.Pageable
import org.springframework.data.domain.PageRequest
import org.springframework.data.domain.Sort
import org.springframework.http.HttpStatus
import org.springframework.stereotype.Service
import org.springframework.transaction.annotation.Transactional
import org.springframework.web.server.ResponseStatusException

@Service
class ContactService(
    private val contactRepository: ContactRepository,
    private val contactMapper: ContactMapper,
) {

    @Transactional
    fun create(request: CreateContactRequest): ContactResponse {
        return save(contactMapper.toEntity(request))
    }

    @Transactional(readOnly = true)
    fun get(id: Long): ContactResponse {
        return contactRepository.findById(id)
            .map(contactMapper::toResponse)
            .orElseThrow { notFound(id) }
    }

    @Transactional(readOnly = true)
    fun list(pageable: Pageable, status: ContactStatus?): ContactPageResponse {
        val effectiveSort = if (pageable.sort.isSorted) pageable.sort else Sort.by(Sort.Order.asc("id"))
        val effectivePageable = PageRequest.of(pageable.pageNumber, pageable.pageSize, effectiveSort)
        val resultPage = if (status == null) {
            contactRepository.findAll(effectivePageable)
        } else {
            contactRepository.findAllByStatus(status, effectivePageable)
        }

        return contactMapper.toPageResponse(resultPage)
    }

    @Transactional
    fun update(id: Long, request: UpdateContactRequest): ContactResponse {
        val entity = contactRepository.findById(id)
            .orElseThrow { notFound(id) }

        contactMapper.updateEntity(request, entity)

        return save(entity)
    }

    @Transactional
    fun delete(id: Long) {
        val entity = contactRepository.findById(id)
            .orElseThrow { notFound(id) }

        contactRepository.delete(entity)
    }

    private fun save(entity: Contact): ContactResponse {
        return contactMapper.toResponse(contactRepository.save(entity))
    }

    private fun notFound(id: Long): ResponseStatusException {
        return ResponseStatusException(HttpStatus.NOT_FOUND, "Contact $id was not found")
    }
}
