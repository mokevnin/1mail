package io.onemail.segment

import io.onemail.account.Account
import io.onemail.model.CreateSegmentRequest
import io.onemail.model.UpdateSegmentRequest
import org.springframework.data.domain.Pageable
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.security.core.context.SecurityContextHolder
import org.springframework.security.oauth2.core.OAuth2AuthenticatedPrincipal
import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.PutMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.server.ResponseStatusException
import io.onemail.model.Segment as SegmentDto

@RestController
class SegmentController(
    private val segmentService: SegmentService,
    private val segmentMapper: SegmentMapper,
) {
    @PostMapping("/api/segments")
    fun segmentsCreate(
        @jakarta.validation.Valid @RequestBody createSegmentRequest: CreateSegmentRequest,
    ): ResponseEntity<SegmentDto> {
        val segment = segmentMapper.fromCreate(createSegmentRequest, account())
        validateDefinition(segment)
        val saved = segmentService.create(account(), segment)
        return ResponseEntity
            .status(HttpStatus.CREATED)
            .body(segmentMapper.toDto(saved))
    }

    @GetMapping("/api/segments")
    fun segmentsList(pageable: Pageable): ResponseEntity<org.springframework.data.domain.Page<SegmentDto>> =
        ResponseEntity.ok(
            segmentService
                .list(account(), pageable)
                .map(segmentMapper::toDto),
        )

    @GetMapping("/api/segments/{id}")
    fun segmentsGet(
        @PathVariable id: String,
    ): ResponseEntity<SegmentDto> {
        val segment =
            segmentService.get(account(), parseId(id))
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Segment not found")
        return ResponseEntity.ok(segmentMapper.toDto(segment))
    }

    @PutMapping("/api/segments/{id}")
    fun segmentsUpdate(
        @PathVariable id: String,
        @jakarta.validation.Valid @RequestBody updateSegmentRequest: UpdateSegmentRequest,
    ): ResponseEntity<SegmentDto> {
        val existing =
            segmentService.get(account(), parseId(id))
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Segment not found")
        segmentMapper.updateFromUpdate(updateSegmentRequest, existing)
        validateDefinition(existing)
        val saved =
            segmentService.update(account(), parseId(id), existing)
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Segment not found")
        return ResponseEntity.ok(segmentMapper.toDto(saved))
    }

    @DeleteMapping("/api/segments/{id}")
    fun segmentsDelete(
        @PathVariable id: String,
    ): ResponseEntity<Void> {
        if (!segmentService.delete(account(), parseId(id))) {
            throw ResponseStatusException(HttpStatus.NOT_FOUND, "Segment not found")
        }
        return ResponseEntity.noContent().build()
    }

    private fun validateDefinition(segment: Segment) {
        if (segment.type == SegmentType.RULE && segment.definition.isNullOrBlank()) {
            throw ResponseStatusException(HttpStatus.UNPROCESSABLE_ENTITY, "Segment definition required for rule type")
        }
    }

    private fun account(): Account {
        val principal = SecurityContextHolder.getContext().authentication!!.principal as OAuth2AuthenticatedPrincipal
        return principal.getAttribute("account")!!
    }

    private fun parseId(id: String): Long =
        try {
            id.toLong()
        } catch (e: IllegalArgumentException) {
            throw ResponseStatusException(HttpStatus.BAD_REQUEST, "Invalid ID format")
        }
}
