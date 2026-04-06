setup:
	./gradlew build 

dev:
	overmind start

test:
	./gradlew test

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

format:
	ktlint --format
