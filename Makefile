# Common tasks. Run "make" with no target to format, vet, test, and build.
BINARY := trimstats
# Strip debug symbols from release builds to make them smaller.
RELEASE_FLAGS := -trimpath -ldflags "-s -w"

.PHONY: all fmt vet test race cover bench build dist compare bootstrap profile r-samples r-time clean

all: fmt vet test build

fmt:            ## reformat code to the standard Go style
	gofmt -s -w .

vet:            ## catch suspicious code the compiler allows
	go vet ./...

test:           ## run every unit test, including the exact Go-vs-R check
	go test -v ./...

race:           ## run the tests with the data race detector (checks the parallel workers)
	go test -race ./...

cover:          ## test coverage report, opened in the browser
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

bench:          ## time the full bootstrap study with 1 worker and with every CPU core
	go test -run '^$$' -bench . -benchmem ./internal/bootstrap

build:          ## executable for this computer
	go build -o $(BINARY) .

dist:           ## executables for Mac (Apple Silicon and Intel) and Windows
	GOOS=darwin  GOARCH=arm64 go build $(RELEASE_FLAGS) -o dist/$(BINARY)-mac-arm64 .
	GOOS=darwin  GOARCH=amd64 go build $(RELEASE_FLAGS) -o dist/$(BINARY)-mac-intel .
	GOOS=windows GOARCH=amd64 go build $(RELEASE_FLAGS) -o dist/$(BINARY).exe .

compare: build  ## print the Go-vs-R comparison table
	./$(BINARY) compare

bootstrap: build  ## run the bootstrap study with the R script's settings
	./$(BINARY) bootstrap

profile: build  ## run the study on one core with CPU and memory profiles, then show the top users
	./$(BINARY) bootstrap -workers 1 -cpuprofile cpu.prof -memprofile mem.prof > /dev/null
	go tool pprof -top -nodecount=10 $(BINARY) cpu.prof
	go tool pprof -sample_index=alloc_space -top -nodecount=5 $(BINARY) mem.prof

r-samples:      ## recreate the samples and R's results in data/ (needs R)
	Rscript r/make-samples.R

r-time: build   ## time the R bootstrap script against the Go version (needs R)
	time Rscript r/run-bootstrap-robust.R > /dev/null
	time ./$(BINARY) bootstrap -workers 1 > /dev/null
	time ./$(BINARY) bootstrap > /dev/null

clean:
	rm -rf $(BINARY) dist coverage.out *.prof
