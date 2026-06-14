.PHONY: dev dev-all stop stop-all logs help docs \
        dev-auth dev-staff dev-student dev-course dev-enrollment dev-attendance dev-grades dev-meal dev-payment \
        stop-auth stop-staff stop-student stop-course stop-enrollment stop-attendance stop-grades stop-meal stop-payment \
        infra infra-down status clean-logs rotate-logs ensure-watchdog \
        gen-types gen-frontend-types gen-mobile-types check-types

# Use bash for proper shell features
SHELL := /bin/bash

# Get absolute path to backend directory
BACKEND_DIR := $(shell pwd)
SERVICES_DIR := $(BACKEND_DIR)/services
LOG_DIR := $(BACKEND_DIR)/tmp/logs

# Service list (HTTP services using air)
SERVICES := auth-service staff-service student-service course-catalog-service enrollment-service attendance-service grades-service meal-service payment-service

# gRPC services (run with go run)
GRPC_SERVICES := 

# Colors
GREEN := \033[0;32m
RED := \033[0;31m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m

# ============================================
# 🚀 MAIN COMMANDS
# ============================================

# Start all services
dev: dev-all

dev-all: ensure-watchdog
	@$(MAKE) rotate-logs --no-print-directory
	@echo -e "$(GREEN)🚀 Starting all backend services...$(NC)"
	@for service in $(SERVICES); do \
		echo -e "$(BLUE)  Starting $$service...$(NC)"; \
		cd $(SERVICES_DIR)/$$service && nohup air > $(LOG_DIR)/$$service.log 2>&1 & echo $$! > $(LOG_DIR)/$$service.pid; \
	done
	@for service in $(GRPC_SERVICES); do \
		echo -e "$(BLUE)  Starting $$service (gRPC)...$(NC)"; \
		cd $(SERVICES_DIR)/$$service && nohup go run ./cmd/main.go > $(LOG_DIR)/$$service.log 2>&1 & echo $$! > $(LOG_DIR)/$$service.pid; \
	done
	@sleep 2
	@echo ""
	@echo -e "$(GREEN)✅ All services started! (log watchdog: max 5MB per service, checks every 10s)$(NC)"
	@echo ""
	@$(MAKE) status --no-print-directory

# Stop all services
stop: stop-all

stop-all:
	@echo -e "$(RED)🛑 Stopping all backend services...$(NC)"
	@# Stop log watchdog
	@if [ -f $(LOG_DIR)/log-watchdog.pid ]; then \
		kill $$(cat $(LOG_DIR)/log-watchdog.pid) 2>/dev/null || true; \
		rm -f $(LOG_DIR)/log-watchdog.pid; \
	fi
	@for service in $(SERVICES) $(GRPC_SERVICES); do \
		if [ -f $(LOG_DIR)/$$service.pid ]; then \
			pid=$$(cat $(LOG_DIR)/$$service.pid); \
			if ps -p $$pid > /dev/null 2>&1; then \
				kill $$pid 2>/dev/null || true; \
				echo "  Stopped $$service (PID: $$pid)"; \
			fi; \
			rm -f $(LOG_DIR)/$$service.pid; \
		fi; \
	done
	@pkill -f "air" 2>/dev/null || true
	@echo -e "$(GREEN)✅ All services stopped$(NC)"

# Restart all services
restart: stop-all dev-all

# Show status of all services
status:
	@echo -e "$(BLUE)📊 Backend Services Status:$(NC)"
	@echo ""
	@printf "%-25s %-10s %-10s\n" "SERVICE" "STATUS" "PID"
	@printf "%-25s %-10s %-10s\n" "-------" "------" "---"
	@for service in $(SERVICES) $(GRPC_SERVICES); do \
		if [ -f $(LOG_DIR)/$$service.pid ]; then \
			pid=$$(cat $(LOG_DIR)/$$service.pid); \
			if ps -p $$pid > /dev/null 2>&1; then \
				printf "%-25s $(GREEN)%-10s$(NC) %-10s\n" "$$service" "running" "$$pid"; \
			else \
				printf "%-25s $(RED)%-10s$(NC) %-10s\n" "$$service" "stopped" "-"; \
			fi; \
		else \
			printf "%-25s $(YELLOW)%-10s$(NC) %-10s\n" "$$service" "not started" "-"; \
		fi; \
	done
	@echo ""

# ============================================
# 📝 LOGS
# ============================================

# Max log file size: 5MB, trim to 3MB when exceeded
LOG_MAX_BYTES := 5242880
LOG_TRIM_BYTES := 3145728

# Start log watchdog if not already running (checks every 10s, trims at 5MB)
ensure-watchdog:
	@mkdir -p $(LOG_DIR)
	@if [ -f $(LOG_DIR)/log-watchdog.pid ] && kill -0 $$(cat $(LOG_DIR)/log-watchdog.pid) 2>/dev/null; then \
		true; \
	else \
		nohup bash -c 'while true; do sleep 10; for f in $(LOG_DIR)/*.log; do \
			[ -f "$$f" ] || continue; \
			s=$$(stat -c%s "$$f" 2>/dev/null || echo 0); \
			[ "$$s" -gt $(LOG_MAX_BYTES) ] && tail -c $(LOG_TRIM_BYTES) "$$f" > "$$f.tmp" && mv "$$f.tmp" "$$f"; \
		done; done' > /dev/null 2>&1 & echo $$! > $(LOG_DIR)/log-watchdog.pid; \
	fi

# Trim all log files exceeding 5MB
rotate-logs:
	@for logfile in $(LOG_DIR)/*.log; do \
		if [ -f "$$logfile" ]; then \
			size=$$(stat -c%s "$$logfile" 2>/dev/null || echo 0); \
			if [ "$$size" -gt $(LOG_MAX_BYTES) ]; then \
				tail -c $(LOG_TRIM_BYTES) "$$logfile" > "$$logfile.tmp" && mv "$$logfile.tmp" "$$logfile"; \
				echo -e "  $(YELLOW)Trimmed $$(basename $$logfile)$(NC) ($$(( size / 1024 ))KB -> $$(( $(LOG_TRIM_BYTES) / 1024 ))KB)"; \
			fi; \
		fi; \
	done

# Delete all log files
clean-logs:
	@echo -e "$(YELLOW)🧹 Cleaning all logs...$(NC)"
	@rm -f $(LOG_DIR)/*.log
	@echo -e "$(GREEN)✅ All logs cleaned$(NC)"

logs:
	@tail -f $(LOG_DIR)/*.log

logs-auth:
	@tail -f $(LOG_DIR)/auth-service.log

logs-staff:
	@tail -f $(LOG_DIR)/staff-service.log

logs-student:
	@tail -f $(LOG_DIR)/student-service.log

logs-course:
	@tail -f $(LOG_DIR)/course-catalog-service.log

logs-enrollment:
	@tail -f $(LOG_DIR)/enrollment-service.log

logs-attendance:
	@tail -f $(LOG_DIR)/attendance-service.log

logs-grades:
	@tail -f $(LOG_DIR)/grades-service.log

logs-meal:
	@tail -f $(LOG_DIR)/meal-service.log

logs-payment:
	@tail -f $(LOG_DIR)/payment-service.log

# ============================================
# 🔧 INDIVIDUAL SERVICES
# ============================================

dev-auth: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting auth-service...$(NC)"
	@cd $(SERVICES_DIR)/auth-service && nohup air > $(LOG_DIR)/auth-service.log 2>&1 & echo $$! > $(LOG_DIR)/auth-service.pid
	@echo -e "$(GREEN)✅ auth-service started (logs: make logs-auth)$(NC)"

dev-staff: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting staff-service...$(NC)"
	@cd $(SERVICES_DIR)/staff-service && nohup air > $(LOG_DIR)/staff-service.log 2>&1 & echo $$! > $(LOG_DIR)/staff-service.pid
	@echo -e "$(GREEN)✅ staff-service started (logs: make logs-staff)$(NC)"

dev-student: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting student-service...$(NC)"
	@cd $(SERVICES_DIR)/student-service && nohup air > $(LOG_DIR)/student-service.log 2>&1 & echo $$! > $(LOG_DIR)/student-service.pid
	@echo -e "$(GREEN)✅ student-service started (logs: make logs-student)$(NC)"

dev-course: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting course-catalog-service...$(NC)"
	@cd $(SERVICES_DIR)/course-catalog-service && nohup air > $(LOG_DIR)/course-catalog-service.log 2>&1 & echo $$! > $(LOG_DIR)/course-catalog-service.pid
	@echo -e "$(GREEN)✅ course-catalog-service started (logs: make logs-course)$(NC)"

dev-enrollment: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting enrollment-service...$(NC)"
	@cd $(SERVICES_DIR)/enrollment-service && nohup air > $(LOG_DIR)/enrollment-service.log 2>&1 & echo $$! > $(LOG_DIR)/enrollment-service.pid
	@echo -e "$(GREEN)✅ enrollment-service started (logs: make logs-enrollment)$(NC)"

dev-attendance: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting attendance-service...$(NC)"
	@cd $(SERVICES_DIR)/attendance-service && nohup air > $(LOG_DIR)/attendance-service.log 2>&1 & echo $$! > $(LOG_DIR)/attendance-service.pid
	@echo -e "$(GREEN)✅ attendance-service started (logs: make logs-attendance)$(NC)"

dev-grades: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting grades-service...$(NC)"
	@cd $(SERVICES_DIR)/grades-service && nohup air > $(LOG_DIR)/grades-service.log 2>&1 & echo $$! > $(LOG_DIR)/grades-service.pid
	@echo -e "$(GREEN)✅ grades-service started (logs: make logs-grades)$(NC)"

dev-meal: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting meal-service...$(NC)"
	@cd $(SERVICES_DIR)/meal-service && nohup air > $(LOG_DIR)/meal-service.log 2>&1 & echo $$! > $(LOG_DIR)/meal-service.pid
	@echo -e "$(GREEN)✅ meal-service started (logs: make logs-meal)$(NC)"

dev-payment: ensure-watchdog
	@echo -e "$(GREEN)🚀 Starting payment-service (gRPC)...$(NC)"
	@cd $(SERVICES_DIR)/payment-service && nohup air > $(LOG_DIR)/payment-service.log 2>&1 & echo $$! > $(LOG_DIR)/payment-service.pid
	@echo -e "$(GREEN)✅ payment-service started on port 50051 (logs: make logs-payment)$(NC)"

# Stop individual services
stop-auth:
	@if [ -f $(LOG_DIR)/auth-service.pid ]; then kill $$(cat $(LOG_DIR)/auth-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/auth-service.pid; fi
	@echo -e "$(RED)🛑 auth-service stopped$(NC)"

stop-staff:
	@if [ -f $(LOG_DIR)/staff-service.pid ]; then kill $$(cat $(LOG_DIR)/staff-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/staff-service.pid; fi
	@echo -e "$(RED)🛑 staff-service stopped$(NC)"

stop-student:
	@if [ -f $(LOG_DIR)/student-service.pid ]; then kill $$(cat $(LOG_DIR)/student-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/student-service.pid; fi
	@echo -e "$(RED)🛑 student-service stopped$(NC)"

stop-course:
	@if [ -f $(LOG_DIR)/course-catalog-service.pid ]; then kill $$(cat $(LOG_DIR)/course-catalog-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/course-catalog-service.pid; fi
	@echo -e "$(RED)🛑 course-catalog-service stopped$(NC)"

stop-enrollment:
	@if [ -f $(LOG_DIR)/enrollment-service.pid ]; then kill $$(cat $(LOG_DIR)/enrollment-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/enrollment-service.pid; fi
	@echo -e "$(RED)🛑 enrollment-service stopped$(NC)"

stop-attendance:
	@if [ -f $(LOG_DIR)/attendance-service.pid ]; then kill $$(cat $(LOG_DIR)/attendance-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/attendance-service.pid; fi
	@echo -e "$(RED)🛑 attendance-service stopped$(NC)"

stop-grades:
	@if [ -f $(LOG_DIR)/grades-service.pid ]; then kill $$(cat $(LOG_DIR)/grades-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/grades-service.pid; fi
	@echo -e "$(RED)🛑 grades-service stopped$(NC)"

stop-meal:
	@if [ -f $(LOG_DIR)/meal-service.pid ]; then kill $$(cat $(LOG_DIR)/meal-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/meal-service.pid; fi
	@echo -e "$(RED)🛑 meal-service stopped$(NC)"

stop-payment:
	@if [ -f $(LOG_DIR)/payment-service.pid ]; then kill $$(cat $(LOG_DIR)/payment-service.pid) 2>/dev/null || true; rm -f $(LOG_DIR)/payment-service.pid; fi
	@echo -e "$(RED)🛑 payment-service stopped$(NC)"

# ============================================
# 🏗️ INFRASTRUCTURE SHORTCUTS
# ============================================

infra:
	@echo -e "$(GREEN)🏗️ Starting infrastructure...$(NC)"
	@cd $(BACKEND_DIR)/infrastructure && $(MAKE) up

infra-down:
	@echo -e "$(RED)🏗️ Stopping infrastructure...$(NC)"
	@cd $(BACKEND_DIR)/infrastructure && $(MAKE) down

# ============================================
# 📚 API DOCS
# ============================================

docs:
	@echo -e "$(GREEN)📚 Serving API docs at http://localhost:8090/swagger-ui.html$(NC)"
	@cd $(BACKEND_DIR)/docs && python3 serve.py 8090

OPENAPI_SPEC := $(BACKEND_DIR)/docs/openapi.yaml
FRONTEND_DIR := $(BACKEND_DIR)/../frontend
MOBILE_DIR := $(BACKEND_DIR)/../mobile

gen-frontend-types:
	@echo -e "$(GREEN)🔧 Generating frontend API types from $(OPENAPI_SPEC)...$(NC)"
	@cd $(FRONTEND_DIR) && bun run gen:api-types

gen-mobile-types:
	@echo -e "$(GREEN)🔧 Generating mobile API types from $(OPENAPI_SPEC)...$(NC)"
	@cd $(MOBILE_DIR) && npm run gen:api-types

gen-types: gen-frontend-types gen-mobile-types
	@echo -e "$(GREEN)✅ API types regenerated for frontend and mobile$(NC)"

check-types: gen-types
	@echo -e "$(BLUE)🔍 Checking for drift between openapi.yaml and generated types...$(NC)"
	@if ! git -C $(BACKEND_DIR)/.. diff --exit-code frontend/src/lib/api-types.ts mobile/types/api-types.ts; then \
		echo -e "$(RED)❌ Generated API types are out of sync with openapi.yaml$(NC)"; \
		echo -e "$(YELLOW)   Run 'make gen-types' and commit the result$(NC)"; \
		exit 1; \
	fi
	@echo -e "$(GREEN)✅ Generated types are in sync with openapi.yaml$(NC)"

# ============================================
# 🆘 HELP
# ============================================

help:
	@echo ""
	@echo -e "$(BLUE)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo -e "$(BLUE)║           MyDreamCampus Backend Makefile Commands              ║$(NC)"
	@echo -e "$(BLUE)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo -e "$(GREEN)🚀 Main Commands:$(NC)"
	@echo "  make dev             Start all services (alias: make dev-all)"
	@echo "  make stop            Stop all services (alias: make stop-all)"
	@echo "  make restart         Restart all services"
	@echo "  make status          Show status of all services"
	@echo ""
	@echo -e "$(YELLOW)📝 Log Commands:$(NC)"
	@echo "  make logs            Tail all service logs"
	@echo "  make rotate-logs     Trim logs exceeding 5MB"
	@echo "  make clean-logs      Delete all log files"
	@echo "  make logs-auth       Tail auth-service logs"
	@echo "  make logs-staff      Tail staff-service logs"
	@echo "  make logs-student    Tail student-service logs"
	@echo "  make logs-course     Tail course-catalog-service logs"
	@echo "  make logs-enrollment Tail enrollment-service logs"
	@echo "  make logs-attendance Tail attendance-service logs"
	@echo "  make logs-grades     Tail grades-service logs"
	@echo "  make logs-meal       Tail meal-service logs"
	@echo "  make logs-payment    Tail payment-service logs"
	@echo ""
	@echo -e "$(BLUE)🔧 Individual Services:$(NC)"
	@echo "  make dev-auth        Start auth-service"
	@echo "  make dev-staff       Start staff-service"
	@echo "  make dev-student     Start student-service"
	@echo "  make dev-course      Start course-catalog-service"
	@echo "  make dev-enrollment  Start enrollment-service"
	@echo "  make dev-attendance  Start attendance-service"
	@echo "  make dev-grades      Start grades-service"
	@echo "  make dev-meal        Start meal-service"
	@echo "  make dev-payment     Start payment-service (gRPC)"
	@echo ""
	@echo -e "$(RED)🛑 Stop Individual:$(NC)"
	@echo "  make stop-auth       Stop auth-service"
	@echo "  make stop-staff      Stop staff-service"
	@echo "  ... (same pattern for other services)"
	@echo ""
	@echo -e "$(GREEN)🏗️ Infrastructure:$(NC)"
	@echo "  make infra           Start infrastructure (postgres, redis, rabbitmq)"
	@echo "  make infra-down      Stop infrastructure"
	@echo ""
	@echo -e "$(BLUE)📚 Documentation:$(NC)"
	@echo "  make docs            Serve API docs (Swagger UI) at http://localhost:8090"
	@echo ""
	@echo -e "$(BLUE)🔧 API Types (OpenAPI → TypeScript):$(NC)"
	@echo "  make gen-types           Regenerate frontend + mobile TS types from openapi.yaml"
	@echo "  make gen-frontend-types  Regenerate only frontend types"
	@echo "  make gen-mobile-types    Regenerate only mobile types"
	@echo "  make check-types         CI helper — fail if generated types drifted"
	@echo ""
