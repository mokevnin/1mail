package io.onemail.config

import io.onemail.model.ProblemDetails
import jakarta.servlet.http.HttpServletRequest
import jakarta.validation.ConstraintViolationException
import org.springframework.core.Ordered
import org.springframework.core.annotation.Order
import org.springframework.dao.DataIntegrityViolationException
import org.springframework.http.HttpStatus
import org.springframework.http.ResponseEntity
import org.springframework.validation.FieldError
import org.springframework.web.bind.MethodArgumentNotValidException
import org.springframework.web.bind.annotation.ExceptionHandler
import org.springframework.web.bind.annotation.RestControllerAdvice
import org.springframework.web.server.ResponseStatusException

@Order(Ordered.HIGHEST_PRECEDENCE)
@RestControllerAdvice
class ApiProblemDetailsExceptionHandler {
    @ExceptionHandler(MethodArgumentNotValidException::class)
    fun handleMethodArgumentNotValid(
        ex: MethodArgumentNotValidException,
        request: HttpServletRequest,
    ): ResponseEntity<ProblemDetails> {
        val errors =
            ex.bindingResult
                .fieldErrors
                .groupBy(FieldError::getField)
                .mapValues { entry -> entry.value.mapNotNull { it.defaultMessage } }
        val problem =
            ProblemDetails()
                .title("Validation error")
                .status(HttpStatus.UNPROCESSABLE_ENTITY.value())
                .detail("Request validation failed")
                .instance(request.requestURI)
                .errors(errors)
        return ResponseEntity.status(HttpStatus.UNPROCESSABLE_ENTITY).body(problem)
    }

    @ExceptionHandler(ConstraintViolationException::class)
    fun handleConstraintViolation(
        ex: ConstraintViolationException,
        request: HttpServletRequest,
    ): ResponseEntity<ProblemDetails> {
        val errors =
            ex.constraintViolations
                .groupBy { it.propertyPath.toString() }
                .mapValues { entry -> entry.value.map { it.message } }
        val problem =
            ProblemDetails()
                .title("Validation error")
                .status(HttpStatus.UNPROCESSABLE_ENTITY.value())
                .detail("Request validation failed")
                .instance(request.requestURI)
                .errors(errors)
        return ResponseEntity.status(HttpStatus.UNPROCESSABLE_ENTITY).body(problem)
    }

    @ExceptionHandler(DataIntegrityViolationException::class)
    fun handleDataIntegrityViolation(
        ex: DataIntegrityViolationException,
        request: HttpServletRequest,
    ): ResponseEntity<ProblemDetails> {
        val problem =
            ProblemDetails()
                .title("Validation error")
                .status(HttpStatus.UNPROCESSABLE_ENTITY.value())
                .detail("Email already exists")
                .instance(request.requestURI)
        return ResponseEntity.status(HttpStatus.UNPROCESSABLE_ENTITY).body(problem)
    }

    @ExceptionHandler(ResponseStatusException::class)
    fun handleResponseStatus(
        ex: ResponseStatusException,
        request: HttpServletRequest,
    ): ResponseEntity<ProblemDetails> {
        val status = ex.statusCode
        val problem =
            ProblemDetails()
                .title(status.toString())
                .status(status.value())
                .detail(ex.reason)
                .instance(request.requestURI)
        return ResponseEntity.status(status).body(problem)
    }
}
