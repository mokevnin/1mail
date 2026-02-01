plugins {
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
        mavenBom("org.springframework.modulith:spring-modulith-bom:2.1.0-SNAPSHOT")
    }
}

dependencies {
    implementation(libs.springBootWeb)
    implementation(libs.springBootDataJpa)
    implementation(libs.springBootValidation)
    implementation("org.springframework:spring-aop")
    implementation("org.jetbrains.kotlin:kotlin-reflect")
    implementation(libs.jacksonKotlin)
    implementation(libs.jacksonNullable)
    implementation(libs.springBootSecurity)
    implementation(libs.springSecurityResourceServer)
    // springdoc-openapi currently not compatible with Spring Boot 4
    implementation(libs.mapstruct)
    implementation("org.springframework.modulith:spring-modulith-starter-core")
    implementation("org.springframework.modulith:spring-modulith-starter-jpa")

    runtimeOnly(libs.h2)
    runtimeOnly(libs.postgresql)

    testImplementation(libs.springBootWebTest)
    testImplementation(libs.kotlinTestJunit5)
    testImplementation("org.instancio:instancio-junit:5.4.1")
    testImplementation("org.springframework.security:spring-security-test")
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
    generatorName.set("spring")
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
            "useSpringBoot3" to "true",
            "useTags" to "true",
            "documentationProvider" to "none",
            "serializationLibrary" to "jackson",
            "openApiNullable" to "true",
            "useBeanValidation" to "true",
            "enumPropertyNaming" to "original",
        ),
    )
}

sourceSets {
    main {
        java.srcDir(layout.buildDirectory.dir("generated/src/main/java"))
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
