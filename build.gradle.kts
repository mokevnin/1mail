plugins {
    alias(libs.plugins.caupain)
    alias(libs.plugins.kotlinJvm)
    alias(libs.plugins.kotlinSpring)
    alias(libs.plugins.kotlinJpa)
    alias(libs.plugins.kotlinKapt)
    alias(libs.plugins.springBoot)
    alias(libs.plugins.dependencyManagement)
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

dependencyManagement {
    imports {
        mavenBom("org.springframework.modulith:spring-modulith-bom:${libs.versions.springModulith.get()}")
    }
}

dependencies {
    implementation(libs.springBootWeb)
    implementation(libs.springBootDataJpa)
    implementation(libs.springBootValidation)
    implementation(libs.springAop)
    implementation(libs.kotlinReflect)
    implementation(libs.jacksonKotlin)
    implementation(libs.jacksonNullable)
    implementation(libs.springBootSecurity)
    implementation(libs.springSecurityResourceServer)
    // springdoc-openapi currently not compatible with Spring Boot 4
    implementation(libs.mapstruct)
    implementation(libs.springModulithStarterCore)
    implementation(libs.springModulithStarterJpa)

    runtimeOnly(libs.h2)
    runtimeOnly(libs.postgresql)

    testImplementation(libs.springBootWebTest)
    testImplementation(libs.kotlinTestJunit5)
    testImplementation(libs.instancioJunit)
    testImplementation(libs.springSecurityTest)
    testRuntimeOnly(libs.junitPlatformLauncher)

    kapt(libs.mapstructProcessor)
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
            "interfaceOnly" to "true",
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

tasks.matching { it.name.startsWith("kapt") }.configureEach {
    dependsOn("openApiGenerate")
}

kapt {
    correctErrorTypes = true
}
