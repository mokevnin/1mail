package io.onemail

import org.instancio.Select
import org.instancio.TargetSelector
import kotlin.reflect.KProperty1
import kotlin.reflect.jvm.javaField

object KSelect {
    fun <T, V> field(property: KProperty1<T, V>): TargetSelector {
        val field = requireNotNull(property.javaField) { "Unable to resolve field for ${property.name}" }
        return Select.field(field.declaringClass, field.name)
    }
}
