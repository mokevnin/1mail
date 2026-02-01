setup:
	./gradlew build 

dev:
	overmind start

test:
	./gradlew test

upgrade:
	./gradlew wrapper --gradle-version latest

generate:
	npx tsp compile typespec/
	./gradlew openApiGenerate

format:
	ktlint --format
