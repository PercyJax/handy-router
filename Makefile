BINARY = handy-router
INSTALL_DIR = /usr/local/bin
CONFIG_DIR = $(HOME)/.config/handy-router
SYSTEMD_DIR = $(HOME)/.config/systemd/user

.PHONY: build install reinstall uninstall enable disable start stop restart status logs
.PHONY: opencode-serve-enable opencode-serve-disable opencode-serve-start opencode-serve-stop opencode-serve-status opencode-serve-logs

build:
	go build -o $(BINARY) .

install: build
	sudo cp $(BINARY) $(INSTALL_DIR)/$(BINARY)
	sudo chmod +x $(INSTALL_DIR)/$(BINARY)
	mkdir -p $(CONFIG_DIR)
	mkdir -p $(SYSTEMD_DIR)
	@if [ ! -f $(CONFIG_DIR)/config.toml ]; then \
		echo ""; \
		echo "Creating default config at:"; \
		echo "  $(CONFIG_DIR)/config.toml"; \
		echo ""; \
		cp config.example.toml $(CONFIG_DIR)/config.toml; \
		echo "LLM Enhancement setup (press Enter to accept defaults):"; \
		echo ""; \
		default_endpoint="https://opencode.ai/zen/go/v1/chat/completions"; \
		printf "  Endpoint [%s]: " "$$default_endpoint"; read endpoint; \
		endpoint="$${endpoint:-$$default_endpoint}"; \
		default_model="deepseek-v4-flash"; \
		printf "  Model [%s]: " "$$default_model"; read model; \
		model="$${model:-$$default_model}"; \
		printf "  API key: "; read api_key; \
		if [ -n "$$endpoint" ]; then \
			sed -i "s|^endpoint = .*|endpoint = \"$$endpoint\"|" $(CONFIG_DIR)/config.toml; \
		fi; \
		if [ -n "$$model" ]; then \
			sed -i "s|^model = .*|model = \"$$model\"|" $(CONFIG_DIR)/config.toml; \
		fi; \
		if [ -n "$$api_key" ]; then \
			sed -i "s|^api_key = .*|api_key = \"$$api_key\"|" $(CONFIG_DIR)/config.toml; \
			echo ""; \
			echo "API key saved."; \
		else \
			echo ""; \
			echo "No API key set. Edit $(CONFIG_DIR)/config.toml to add it later."; \
		fi; \
	else \
		echo "Config already exists at $(CONFIG_DIR)/config.toml — skipping."; \
	fi
	cp handy-router.service $(SYSTEMD_DIR)/handy-router.service
	cp opencode-serve.service $(SYSTEMD_DIR)/opencode-serve.service
	systemctl --user daemon-reload
	@echo ""
	@echo "Installed!"
	@echo ""
	@echo "  Binary:  $(INSTALL_DIR)/$(BINARY)"
	@echo "  Config:  $(CONFIG_DIR)/config.toml"
	@echo "  Services:"
	@echo "    handy-router.service      (speech-to-text routing)"
	@echo "    opencode-serve.service    (opencode headless server)"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Edit config if needed:  nano $(CONFIG_DIR)/config.toml"
	@echo "  2. Enable both services:   make enable && make opencode-serve-enable"
	@echo "  3. Configure Handy:"
	@echo "     - Provider:   Custom"
	@echo "     - Base URL:   http://localhost:11341/v1"
	@echo "     - Model:      handy-router"
	@echo "     - Prompt:     \$\{output\}"
	@echo ""

reinstall: uninstall install

uninstall: disable opencode-serve-disable
	sudo rm -f $(INSTALL_DIR)/$(BINARY)
	rm -f $(SYSTEMD_DIR)/handy-router.service
	rm -f $(SYSTEMD_DIR)/opencode-serve.service
	systemctl --user daemon-reload
	@echo ""
	@echo "Uninstalled. Config kept at $(CONFIG_DIR)"
	@echo "To remove config: rm -rf $(CONFIG_DIR)"
	@echo ""

# handy-router service
enable:
	systemctl --user enable handy-router.service
	systemctl --user start handy-router.service
	@echo "Service enabled and started."
	@echo "Check status: make status"
	@echo "Check logs:   journalctl --user -u handy-router -f"

disable:
	-systemctl --user stop handy-router.service 2>/dev/null
	-systemctl --user disable handy-router.service 2>/dev/null

start:
	systemctl --user start handy-router.service

stop:
	systemctl --user stop handy-router.service

restart:
	systemctl --user restart handy-router.service

status:
	systemctl --user status handy-router.service

logs:
	journalctl --user -u handy-router -f

# opencode-serve service
opencode-serve-enable:
	systemctl --user enable opencode-serve.service
	systemctl --user start opencode-serve.service
	@echo "opencode-serve enabled and started on port 11342."

opencode-serve-disable:
	-systemctl --user stop opencode-serve.service 2>/dev/null
	-systemctl --user disable opencode-serve.service 2>/dev/null

opencode-serve-start:
	systemctl --user start opencode-serve.service

opencode-serve-stop:
	systemctl --user stop opencode-serve.service

opencode-serve-restart:
	systemctl --user restart opencode-serve.service

opencode-serve-status:
	systemctl --user status opencode-serve.service

opencode-serve-logs:
	journalctl --user -u opencode-serve -f
