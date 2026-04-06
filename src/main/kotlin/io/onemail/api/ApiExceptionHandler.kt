package io.onemail.api

import io.onemail.model.ProblemDetails
import jakarta.servlet.http.HttpServletRequest
import org.springframework.dao.DataIntegrityViolationException
import org.springframework.http.HttpStatus
import org.springframework.http.MediaType
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.ExceptionHandler
import org.springframework.web.bind.annotation.RestControllerAdvice

@RestControllerAdvice
class ApiExceptionHandler {

    @ExceptionHandler(DataIntegrityViolationException::class)
    fun handleDataIntegrityViolation(
        exception: DataIntegrityViolationException,
        request: HttpServletRequest,
    ): ResponseEntity<ProblemDetails> {
        val body = ProblemDetails(
            type = "https://onemail.dev/problems/conflict",
            title = "Conflict",
            status = HttpStatus.CONFLICT.value(),
            detail = "Request conflicts with existing data",
            instance = request.requestURI,
        )

        return ResponseEntity.status(HttpStatus.CONFLICT)
            .contentType(MediaType.APPLICATION_PROBLEM_JSON)
            .body(body)
    }
}
