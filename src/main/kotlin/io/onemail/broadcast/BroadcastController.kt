package io.onemail.broadcast

import io.onemail.account.Account
import io.onemail.model.CreateBroadcastRequest
import io.onemail.model.UpdateBroadcastRequest
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
import io.onemail.model.Broadcast as BroadcastDto

@RestController
class BroadcastController(
    private val broadcastService: BroadcastService,
    private val broadcastMapper: BroadcastMapper,
) {
    @PostMapping("/api/broadcasts")
    fun broadcastsCreate(
        @jakarta.validation.Valid @RequestBody createBroadcastRequest: CreateBroadcastRequest,
    ): ResponseEntity<BroadcastDto> {
        val broadcast = broadcastMapper.fromCreate(createBroadcastRequest, account())
        val saved = broadcastService.create(account(), broadcast, createBroadcastRequest.segmentId)
        return ResponseEntity
            .status(HttpStatus.CREATED)
            .body(broadcastMapper.toDto(saved))
    }

    @GetMapping("/api/broadcasts")
    fun broadcastsList(pageable: Pageable): ResponseEntity<org.springframework.data.domain.Page<BroadcastDto>> =
        ResponseEntity.ok(
            broadcastService
                .list(account(), pageable)
                .map(broadcastMapper::toDto),
        )

    @GetMapping("/api/broadcasts/{id}")
    fun broadcastsGet(
        @PathVariable id: String,
    ): ResponseEntity<BroadcastDto> {
        val broadcast =
            broadcastService.get(account(), parseId(id))
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Broadcast not found")
        return ResponseEntity.ok(broadcastMapper.toDto(broadcast))
    }

    @PutMapping("/api/broadcasts/{id}")
    fun broadcastsUpdate(
        @PathVariable id: String,
        @jakarta.validation.Valid @RequestBody updateBroadcastRequest: UpdateBroadcastRequest,
    ): ResponseEntity<BroadcastDto> {
        val existing =
            broadcastService.get(account(), parseId(id))
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Broadcast not found")
        broadcastMapper.updateFromUpdate(updateBroadcastRequest, existing)
        val segmentId = updateBroadcastRequest.segmentId?.orElse(null)
        val saved =
            broadcastService.update(account(), parseId(id), existing, segmentId)
                ?: throw ResponseStatusException(HttpStatus.NOT_FOUND, "Broadcast not found")
        return ResponseEntity.ok(broadcastMapper.toDto(saved))
    }

    @DeleteMapping("/api/broadcasts/{id}")
    fun broadcastsDelete(
        @PathVariable id: String,
    ): ResponseEntity<Void> {
        if (!broadcastService.delete(account(), parseId(id))) {
            throw ResponseStatusException(HttpStatus.NOT_FOUND, "Broadcast not found")
        }
        return ResponseEntity.noContent().build()
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
