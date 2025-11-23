Overall, you must follow this workflow (reference: ci.yml)

go mod tidy

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v8
        with:
          version: v2.6.1

      - name: Run tests with coverage
        run: go test -v -coverprofile=coverage.out -covermode=atomic ./...

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          flags: unittests
          name: codecov-gocreator

      - name: Build gocreator
        run: go build ./cmd/gocreator
        