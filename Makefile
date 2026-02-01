setup:
	./gradlew build 

dev:
	overmind start

test:
	./gradlew test

upgrade:
	./gradlew wrapper --gradle-version latest
