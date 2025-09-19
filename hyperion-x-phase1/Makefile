REG?=hyperion
TAG?=phase1

# Add 'scan' to the list of phony targets
.PHONY: docker-build-all docker-scheduler docker-controller docker-agent scan

docker-build-all: docker-scheduler docker-controller docker-agent

docker-scheduler:
	docker build -f Dockerfile.scheduler -t $(REG)/scheduler:$(TAG) .

docker-controller:
	docker build -f Dockerfile.controller -t $(REG)/controller:$(TAG) .

docker-agent:
	docker build -f agent.Dockerfile -t $(REG)/agent:$(TAG) ./agent

# New target to scan images for vulnerabilities using trivy
# Assumes trivy is installed: https://github.com/aquasecurity/trivy
scan: docker-build-all
	@echo "\n➡️  Scanning container images for vulnerabilities..."
	@trivy image --severity HIGH,CRITICAL --exit-code 0 $(REG)/scheduler:$(TAG)
	@trivy image --severity HIGH,CRITICAL --exit-code 0 $(REG)/controller:$(TAG)
	@trivy image --severity HIGH,CRITICAL --exit-code 0 $(REG)/agent:$(TAG)