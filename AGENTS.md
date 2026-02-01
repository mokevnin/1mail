# Repository Guidelines

## Project Structure & Module Organization
- `src/main/kotlin/` contains the Spring Boot application code (entry point at `src/main/kotlin/com/example/_mail/DemoApplication.kt`).
- `src/test/kotlin/` holds JVM tests (JUnit 5).
- `src/main/resources/` includes server resources; static assets live under `src/main/resources/static/` and templates under `src/main/resources/templates/`.
- `client/` is the Vite + React frontend (entry `client/main.tsx`, styles `client/main.css`, Vite root set in `vite.config.ts`).
- Build outputs are generated into `build/` by Gradle.

## Build, Test, and Development Commands
- `make setup`: Runs a full Gradle build (`./gradlew build`).
- `make test`: Executes JVM tests via Gradle (`./gradlew test`).
- `./gradlew bootRun`: Starts the Spring Boot API locally.
- `pnpm vite`: Runs the Vite dev server for the frontend (see `Procfile`).
- `make dev`: Uses `overmind start` to run `Procfile` processes (`api` + `web`) together.
- `make upgrade`: Updates the Gradle wrapper to the latest version.

## Coding Style & Naming Conventions
- Kotlin follows standard Kotlin style; keep indentation consistent with existing files (tabs are currently used in `DemoApplication.kt`).
- Frontend code uses 2-space indentation (see `client/main.tsx`).
- No repository-wide formatter or linter is configured yet; keep changes consistent with surrounding code.
- Use clear, descriptive names for classes and components (e.g., `DemoApplication`, `MainLayout`).

## Testing Guidelines
- JVM tests live under `src/test/kotlin/` and run on JUnit 5 (`./gradlew test`).
- No frontend test runner is configured; if you add one, document the command here.
- Name tests to mirror the class or behavior they cover (e.g., `UserServiceTest`).

## Commit & Pull Request Guidelines
- Git history is minimal; no established commit message convention yet. Use short, imperative summaries (e.g., "Add mail list endpoint").
- PRs should include: a clear description, test results (or why tests were skipped), and screenshots for UI changes.
- Link related issues when applicable.

## Configuration Notes
- Application settings live in `application.yml`. Avoid committing secrets; prefer environment variables or local overrides for sensitive values.
