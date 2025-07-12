package server

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (s *Server) startDataPlane() {
	s.logger.LogInfoMessage().Msgf("Data plane starting on %s", s.conf.DataPlaneAddr)

	// The server's handler is our custom proxy.
	server := &http.Server{
		Addr:    s.conf.DataPlaneAddr,
		Handler: http.HandlerFunc(s.proxyHandler),
	}

	if err := server.ListenAndServe(); err != nil {
		s.logger.LogFatalMessage().Err(err).Msg("Data plane failed to start")
	}
}

func (s *Server) proxyHandler(w http.ResponseWriter, r *http.Request) {
	// extract subdomain from requst header
	hostParts := strings.Split(r.Host, ".")
	if len(hostParts) < 3 {
		http.Error(w, "Invalid host format", http.StatusBadRequest)
		return
	}
	clientID := hostParts[0]
	s.logger.LogInfoMessage().Msgf("hostparts: %+v", hostParts)

	// lookup for client in our registry
	s.mutex.Lock()
	client, found := s.clients[clientID]
	s.mutex.Unlock()

	if !found {
		s.logger.LogErrorMessage().Msgf("subdomain for client_id: %v not found in the regisry, or client has disconnected", clientID)
		http.Error(w, fmt.Sprintf("subdomain for client_id: %v not found in the regisry, or client has disconnected", clientID), http.StatusNotFound)
		return
	}

	// Open a new stream to the client over its yamux session.
	// This is a new, independent data channel.
	proxyStream, err := client.session.OpenStream()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to open stream for client_id: %v, err: %v", clientID, err), http.StatusInternalServerError)
		s.logger.LogErrorMessage().Msgf("failed to open stream for client_id: %v, err: %v", clientID, err)
		return
	}
	defer proxyStream.Close()

	s.logger.LogInfoMessage().Str("host", r.Host).Str("path", r.URL.Path).Msg("Proxying request")

	// Forward the public user's request to the client via the stream.
	if err := r.Write(proxyStream); err != nil {
		s.logger.LogErrorMessage().Err(err).Msgf("Failed to write request to proxy stream for client %s", clientID)
		return
	}

	// Read the response from the client via the stream and forward it back to the public user.
	resp, err := http.ReadResponse(bufio.NewReader(proxyStream), r)
	if err != nil {
		// This can happen if the client-side service is down. Send a Bad Gateway error.
		http.Error(w, "Error reading response from client service", http.StatusBadGateway)
		s.logger.LogErrorMessage().Err(err).Msgf("Failed to read response from proxy stream for client %s", clientID)
		return
	}
	defer resp.Body.Close()

	// Copy headers from the client's response to our response writer.
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write the status code and the response body.
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
