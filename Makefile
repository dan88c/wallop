REGISTRY ?= config/tool_registry.yaml
GO_DIR   := core-operator
PY       ?= python3
BIN      := bin/wallop
PREFIX   ?= $(HOME)/.local

.PHONY: help toc validate guard graph register test test-go test-py factory build install tidy sanitize demo bootstrap doctor download

help:
	@echo "wallop targets:"
	@echo "  make build      compile bin/wallop (Go required)"
	@echo "  make download   fetch nightly binary into bin/ (no Go)"
	@echo "  make toc"
	@echo "  make guard TOOL=name PAYLOAD='{...}'"
	@echo "  make graph VAULT=./sandbox/vault QUERY=home"
	@echo "  make register ENTRY=./x.py NAME=my_counter"
	@echo "  make doctor     registry + schema + env health check"
	@echo "  make demo"

bootstrap:
	$(PY) scripts/bootstrap.py

download:
	$(PY) scripts/bootstrap.py --download

build:
	mkdir -p bin $(GO_DIR)/bin
	cd $(GO_DIR) && go build -o ../bin/wallop ./cmd/harness
	cd $(GO_DIR) && go build -o bin/harness ./cmd/harness

install: build
	mkdir -p $(PREFIX)/bin
	cp $(BIN) $(PREFIX)/bin/wallop
	@echo "installed $(PREFIX)/bin/wallop"

toc: build
	$(BIN) toc --registry $(REGISTRY)

validate: build
	@test -n "$(CALL)" || (echo "usage: make validate CALL='{...}'"; exit 1)
	$(BIN) validate --registry $(REGISTRY) --call '$(CALL)'

guard: build
	@if [ -n "$(TOOL)" ]; then \
		$(BIN) guard --registry $(REGISTRY) --tool '$(TOOL)' --payload '$(PAYLOAD)'; \
	elif [ -n "$(CALL)" ]; then \
		$(BIN) guard --registry $(REGISTRY) --call '$(CALL)'; \
	else \
		echo "usage: make guard TOOL=calendar_gateway PAYLOAD='{\"action\":\"list\"}'"; exit 2; \
	fi

register: build
	@test -n "$(ENTRY)" && test -n "$(NAME)" || (echo "usage: make register ENTRY=./script.py NAME=my_counter"; exit 1)
	$(BIN) register --registry $(REGISTRY) --entry '$(ENTRY)' --name '$(NAME)' --desc '$(DESC)' $(if $(TAG),--tag '$(TAG)',) $(if $(PARAM),--param '$(PARAM)',)

sanitize:
	@test -n "$(TEXT)" || (echo "usage: make sanitize TEXT='...'"; exit 1)
	$(PY) -c "import sys; sys.path.insert(0,'developer-factory'); from privacy_guard import sanitize; r=sanitize('''$(TEXT)'''); print(r.abstract); print('redactions', r.redactions, 'blocked', r.blocked)"

graph: build
	@test -n "$(VAULT)$(WIKI)" || (echo "usage: make graph VAULT=./sandbox/vault QUERY=home"; exit 1)
	$(BIN) graph --vault $(if $(VAULT),$(VAULT),$(WIKI)) --query '$(QUERY)'

doctor:
	$(PY) scripts/doctor.py --registry $(REGISTRY)

test: test-go test-py

test-go:
	cd $(GO_DIR) && go test ./...
	cd tests/go_tests && go test ./...

test-py:
	$(PY) -m pytest tests/python_tests tests/test_factory_pipeline.py -q

factory:
	$(PY) developer-factory/factory_agent.py $(ARGS)

tidy:
	cd $(GO_DIR) && go mod tidy

demo:
	MOCK_MODE=true VAULT_PATH=./sandbox/vault $(PY) scripts/demo.py
