.PHONY: docker-build docker-push docker-run docker-stop docker-logs smoke-test k8s-apply k8s-delete port-forward test clean

# --- Configuration ---
PROJECT_ID ?= your-gcp-project-id
REGISTRY   ?= gcr.io/$(PROJECT_ID)
TAG        ?= latest

# --- Docker ---
docker-build:
	docker build -t $(REGISTRY)/ledger:$(TAG)    -f ledger/Dockerfile .
	docker build -t $(REGISTRY)/core:$(TAG)      -f core/Dockerfile .
	docker build -t $(REGISTRY)/processor:$(TAG)  -f processor/Dockerfile .
	docker build -t $(REGISTRY)/webapp:$(TAG)     -f webapp/Dockerfile .

docker-push:
	docker push $(REGISTRY)/ledger:$(TAG)
	docker push $(REGISTRY)/core:$(TAG)
	docker push $(REGISTRY)/processor:$(TAG)
	docker push $(REGISTRY)/webapp:$(TAG)

# --- Local Docker (development) ---
docker-run: docker-stop
	docker network create banking-net 2>/dev/null || true
	docker run -d --name ledger    --network banking-net -p 8080:8080 -v banking-data:/app/data ledger:$(TAG)
	docker run -d --name core      --network banking-net -p 8081:8081 -e LEDGER_URL=http://ledger:8080 core:$(TAG)
	docker run -d --name processor --network banking-net -p 8082:8082 -e CORE_URL=http://core:8081 processor:$(TAG)
	docker run -d --name webapp    --network banking-net -p 8083:8083 -e LEDGER_URL=http://ledger:8080 -e PROCESSOR_URL=http://processor:8082 webapp:$(TAG)
	@echo "\n✅ All services running — ledger :8080, core :8081, processor :8082, webapp :8083"

docker-stop:
	docker rm -f ledger core processor webapp 2>/dev/null || true
	docker network rm banking-net 2>/dev/null || true
	@echo "✅ All services stopped"

docker-logs:
	docker logs -f ledger & docker logs -f core & docker logs -f processor & docker logs -f webapp

smoke-test:
	bash scripts/smoke_test.sh

# --- Kubernetes ---
k8s-apply:
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/ledger.yaml
	kubectl apply -f k8s/core.yaml
	kubectl apply -f k8s/processor.yaml
	kubectl apply -f k8s/webapp.yaml

k8s-delete:
	kubectl delete -f k8s/webapp.yaml     --ignore-not-found
	kubectl delete -f k8s/processor.yaml  --ignore-not-found
	kubectl delete -f k8s/core.yaml       --ignore-not-found
	kubectl delete -f k8s/ledger.yaml     --ignore-not-found
	kubectl delete -f k8s/namespace.yaml  --ignore-not-found

port-forward:
	@echo "Forwarding ledger :8080 and processor :8082 to localhost..."
	@echo "Run each in a separate terminal:"
	@echo "  kubectl port-forward -n core-banking svc/ledger 8080:8080"
	@echo "  kubectl port-forward -n core-banking svc/processor 8082:8082"

# --- Local development ---
test:
	go build ./...
	go test ./...

clean:
	rm -rf data/
