#!/bin/bash
set -e # Exit immediately if a command exits with a non-zero status.

# --- Helper Functions for a Professional-Looking Output ---
step_header() {
    printf "\n\n"
    printf "=======================================================================\n"
    printf "  STEP %-5s: %s\n" "$1" "$2"
    printf "=======================================================================\n"
}

info() {
    printf "🔹 %s\n" "$1"
}

success() {
    printf "✅ %s\n" "$1"
}

code_block() {
    # Prints the output in a distinct color to make it stand out
    printf "    \e[33m%s\e[0m\n" "$1"
}

# --- 0. SCRIPT SETUP AND CLEANUP ---
step_header "0" "Cleanup & Preparation"
info "Cleaning up any previous demo runs..."
kubectl delete aijob --all -n default --ignore-not-found >/dev/null 2>&1
kubectl wait --for=delete pod -l app=hyperion-aijob -n default --timeout=60s || true
pkill -f "kubectl port-forward" || true
pkill -f "kubectl logs -f" || true
success "Environment is clean."

# --- 1. BUILD, LOAD & DEPLOY ---
step_header "1" "Build, Load & Deploy Hyperion Control Plane"
info "Building images directly inside Minikube for reliability..."
eval $(minikube -p minikube docker-env)
make docker-build-all

info "Deploying and verifying Hyperion system components..."
kubectl rollout restart deployment/hyperion-controller -n hyperion-system
kubectl rollout status deployment/hyperion-controller -n hyperion-system --timeout=180s

kubectl rollout restart deployment/hyperion-scheduler -n hyperion-system
kubectl rollout status deployment/hyperion-scheduler -n hyperion-system --timeout=180s

kubectl rollout restart daemonset/hyperion-agent -n hyperion-system
kubectl rollout status daemonset/hyperion-agent -n hyperion-system --timeout=180s
success "Hyperion control plane is fully operational and ready."

# --- 2. SHOWCASE THE HYPERION AGENT ---
step_header "2" "Showcase: The High-Fidelity Rust Agent"
info "Finding an agent pod to inspect..."
AGENT_POD=$(kubectl get pods -n hyperion-system -l app=hyperion-agent -o jsonpath='{.items[0].metadata.name}')
success "Found agent pod: $AGENT_POD"

info "Forwarding port 9090 to the agent pod..."
kubectl port-forward -n hyperion-system "$AGENT_POD" 9090:9090 &
PF_AGENT_PID=$!
sleep 3

info "Querying the /metrics endpoint to show live, custom telemetry..."
if command -v jq &> /dev/null; then
    curl -s http://localhost:9090/metrics | jq
else
    curl -s http://localhost:9090/metrics
fi
echo
success "Live metrics successfully retrieved from the Rust agent."
kill $PF_AGENT_PID

# --- 3. SHOWCASE THE INTELLIGENT SCHEDULER ---
step_header "3" "Showcase: The Intelligent Go Scheduler"
info "Finding a scheduler pod to observe..."
SCHEDULER_POD=$(kubectl get pods -n hyperion-system -l app=hyperion-scheduler -o jsonpath='{.items[0].metadata.name}')
success "Found scheduler pod: $SCHEDULER_POD"

info "Streaming the scheduler's logs in real-time to watch its decisions..."
kubectl logs -f "$SCHEDULER_POD" -n hyperion-system &
LOGS_PID=$!
sleep 3

info "\nSubmitting a new AIJob with parallelism=2..."
kubectl apply -f deploy/k8s/crd/samples/aijob-bert.yaml

info "\nWatching the scheduler logs above for scoring and binding decisions..."
kubectl wait --for=condition=Initialized pod -l hyperion.ai/aijob=aijob-bert-small -n default --timeout=60s
success "Controller has created the pods."

kubectl wait --for=jsonpath='{.spec.nodeName}' pod -l hyperion.ai/aijob=aijob-bert-small -n default --timeout=60s
success "Scheduler has successfully assigned all pods!"

info "Pausing for 5 seconds to allow for observation of the logs..."
sleep 5

# --- 4. THE FINAL PROOF ---
step_header "4" "Final Proof of Scheduling"
info "Inspecting the newly scheduled pods to confirm Node assignment..."
POD_TO_INSPECT=$(kubectl get pods -n default -l hyperion.ai/aijob=aijob-bert-small -o jsonpath='{.items[0].metadata.name}')
code_block "Pod: $POD_TO_INSPECT"
code_block "Node: $(kubectl get pod $POD_TO_INSPECT -n default -o jsonpath='{.spec.nodeName}')"
success "Scheduling confirmed via pod spec."

# --- 5. CLEANUP ---
step_header "5" "Cleanup"
info "Stopping the scheduler log stream..."
kill $LOGS_PID
info "Deleting the AIJob..."
kubectl delete aijob aijob-bert-small -n default
success "Demo resources cleaned up."

# --- FINAL SUMMARY ---
printf "\n\n"
echo "-------------------------------------"
success "Hyperion-X Phase 1 Showcase Completed!"
echo "-------------------------------------"