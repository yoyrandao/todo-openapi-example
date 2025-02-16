.PHONY: build-openapi
build-openapi:
	docker run --rm -v ./openapi:/spec redocly/cli bundle /spec/spec.yaml --output /spec/spec.json