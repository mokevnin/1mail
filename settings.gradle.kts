rootProject.name = "1mail"

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.PREFER_SETTINGS)
    repositories {
        maven { url = uri("https://repo.spring.io/milestone") }
        maven { url = uri("https://repo.spring.io/snapshot") }
        mavenCentral()
    }
    versionCatalogs {
        create("libs") {
            version("kotlin", "2.2.21")
            version("springBoot", "4.0.2")
            version("dependencyManagement", "1.1.7")
            version("openapiGenerator", "7.14.0")
            version("mapstruct", "1.5.5.Final")
            version("jackson", "2.17.2")
            version("jacksonNullable", "0.2.6")

            library("jacksonKotlin", "com.fasterxml.jackson.module", "jackson-module-kotlin").versionRef("jackson")
            library("jacksonNullable", "org.openapitools", "jackson-databind-nullable").versionRef("jacksonNullable")
            library("mapstruct", "org.mapstruct", "mapstruct").versionRef("mapstruct")
            library("mapstructProcessor", "org.mapstruct", "mapstruct-processor").versionRef("mapstruct")

            library("springBootWeb", "org.springframework.boot", "spring-boot-starter-web").withoutVersion()
            library("springBootDataJpa", "org.springframework.boot", "spring-boot-starter-data-jpa").withoutVersion()
            library("springBootValidation", "org.springframework.boot", "spring-boot-starter-validation").withoutVersion()
            library("springBootAop", "org.springframework.boot", "spring-boot-starter-aop").withoutVersion()
            library("springBootSecurity", "org.springframework.boot", "spring-boot-starter-security").withoutVersion()
            library(
                "springSecurityResourceServer",
                "org.springframework.security",
                "spring-security-oauth2-resource-server",
            ).withoutVersion()

            library("h2", "com.h2database", "h2").withoutVersion()
            library("postgresql", "org.postgresql", "postgresql").withoutVersion()

            library("springBootWebTest", "org.springframework.boot", "spring-boot-starter-webmvc-test").withoutVersion()
            library("kotlinTestJunit5", "org.jetbrains.kotlin", "kotlin-test-junit5").withoutVersion()
            library("junitPlatformLauncher", "org.junit.platform", "junit-platform-launcher").withoutVersion()

            plugin("kotlinJvm", "org.jetbrains.kotlin.jvm").versionRef("kotlin")
            plugin("kotlinSpring", "org.jetbrains.kotlin.plugin.spring").versionRef("kotlin")
            plugin("kotlinJpa", "org.jetbrains.kotlin.plugin.jpa").versionRef("kotlin")
            plugin("kotlinKapt", "org.jetbrains.kotlin.kapt").versionRef("kotlin")
            plugin("springBoot", "org.springframework.boot").versionRef("springBoot")
            plugin("dependencyManagement", "io.spring.dependency-management").versionRef("dependencyManagement")
            plugin("openapiGenerator", "org.openapi.generator").versionRef("openapiGenerator")
        }
    }
}
