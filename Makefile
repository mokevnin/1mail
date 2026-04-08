setup:
	./gradlew build 

dev:
	overmind start

test:
	./gradlew test

update: update-gradle update-gradle-deps update-npm-deps

update-gradle:
	./gradlew wrapper --gradle-version latest

update-npm-deps:
	npx ncu -u
	pnpm update

update-gradle-deps:
	./gradlew replaceOutdatedDependencies --no-configuration-cache

generate:
	npx tsp compile typespec/
	./gradlew openApiGenerate
	./gradlew clean openApiGenerate

format:
	ktlint --format
