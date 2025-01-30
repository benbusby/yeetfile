.PHONY: backend web cli

web:
	@echo -----------------------------------------
	@echo "   Updating submodules..."
	@echo -----------------------------------------
	git submodule update --init --recursive
	@echo -----------------------------------------
	@echo "   Generating TypeScript files from go..."
	@echo -----------------------------------------
	go run utils/generate_typescript.go ./web/ts
	@echo -----------------------------------------
	@echo "   Compiling TypeScript to JavaScript..."
	@echo -----------------------------------------
	tsc --removeComments

backend: web
	@echo -----------------------------------------
	@echo "   Building backend..."
	@echo -----------------------------------------
	go build -ldflags="-s -w" -tags yeetfile-server -o yeetfile-server ./backend
	@echo -----------------------------------------
	@echo "   Build complete: ./yeetfile-server"
	@echo -----------------------------------------

cli:
	go build -ldflags="-s -w" -tags yeetfile -o yeetfile ./cli

clean:
	rm -f yeetfile-web
	rm -f yeetfile
