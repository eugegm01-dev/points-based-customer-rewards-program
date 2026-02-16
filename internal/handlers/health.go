package handlers

import "net/http"

// Health responds with 200 OK for liveness/readiness probes.
// Used by load balancers and orchestrators (e.g. Kubernetes).
func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}
