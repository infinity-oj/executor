.PHONY: clean
clean:
	- rm executor

executor: clean
	go build -o executor ./cmd/executor
