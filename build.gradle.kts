plugins {
    alias(libs.plugins.caupain)
    alias(libs.plugins.dependencyManagement)
    alias(libs.plugins.kotlinJvm)
    alias(libs.plugins.kotlinSpring)
    alias(libs.plugins.springBoot)
    alias(libs.plugins.openapiGenerator)
}

group = "io.onemail"
version = "0.0.1-SNAPSHOT"
description = "Open source email marketing platform"

java {
    toolchain {
        languageVersion = JavaLanguageVersion.of(21)
    }
}

dependencies {
    implementation(libs.springBootWeb)
    implementation(libs.springBootValidation)
    implementation(libs.kotlinReflect)
    implementation(libs.jacksonKotlin)

    testImplementation(libs.springBootWebTest)
    testImplementation(libs.kotlinTestJunit5)
    testRuntimeOnly(libs.junitPlatformLauncher)
}

kotlin {
    compilerOptions {
        freeCompilerArgs.addAll("-Xjsr305=strict", "-Xannotation-default-target=param-property")
    }
}

tasks.withType<Test> {
    useJUnitPlatform()
    testLogging {
        events("failed", "standardOut", "standardError")
        exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL
        showStandardStreams = true
    }
}

// OpenAPI Generator
openApiGenerate {
    generatorName.set("kotlin-spring")
    inputSpec.set("$projectDir/openapi/openapi.yaml")
    outputDir.set(
        layout.buildDirectory
            .dir("generated")
            .get()
            .asFile.absolutePath,
    )
    apiPackage.set("io.onemail.api")
    modelPackage.set("io.onemail.model")
    configOptions.set(
        mapOf(
            "delegatePattern" to "true",
            "useSpringBoot4" to "true",
            "useTags" to "true",
            "documentationProvider" to "none",
            "serializationLibrary" to "jackson",
            "openApiNullable" to "true",
            "useJackson3" to "false",
            "useBeanValidation" to "true",
            "enumPropertyNaming" to "original",
        ),
    )
}

sourceSets {
    main {
        kotlin.srcDir(layout.buildDirectory.dir("generated/src/main/kotlin"))
    }
}

tasks.named("compileKotlin") {
    dependsOn("openApiGenerate")
}
