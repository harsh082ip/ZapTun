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
	// 1. Extract the full tunnel ID (the entire subdomain) from the host.
	hostParts := strings.Split(r.Host, ".")
	if len(hostParts) < 3 {
		http.Error(w, "Invalid host format", http.StatusBadRequest)
		return
	}
	tunnelID := hostParts[0]

	// [note]: I am using nginx, that is routing user's request to my server, so all requests come from localhost.
	// To get the actual user's IP, I need to read it from the X-Forwarded-For header set by Nginx.
	// if u dont wish to use that, fallback will work for you
	userIP := r.Header.Get("X-Forwarded-For")
	if userIP == "" {
		userIP = strings.Split(r.RemoteAddr, ":")[0] // fallback
	}

	r.Header.Set("X-Forwarded-For", userIP)

	// 2. Extract the base username from the tunnel ID.
	// This works for both "username" and "username-1".
	userLogin := strings.Split(tunnelID, "-")[0]

	// 3. Perform the two-step lookup in the new 'users' map.
	s.mutex.RLock()
	userRecord, userFound := s.users[userLogin]
	var client *Client
	var tunnelFound bool
	if userFound {
		// If the user exists, look for their specific tunnel.
		client, tunnelFound = userRecord.tunnels[tunnelID]
	}
	s.mutex.RUnlock()

	if !tunnelFound {
		msg := fmt.Sprintf("subdomain for client_id: %v not found in the registry, or client has disconnected", tunnelID)
		s.logger.LogErrorMessage().Msg(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	proxyStream, err := client.session.OpenStream()
	if err != nil {
		msg := fmt.Sprintf("failed to open stream for client_id: %v, err: %v", tunnelID, err)
		http.Error(w, msg, http.StatusInternalServerError)
		s.logger.LogErrorMessage().Msg(msg)
		return
	}
	defer proxyStream.Close()

	s.logger.LogInfoMessage().Str("host", r.Host).Str("path", r.URL.Path).Msg("Proxying request")

	if err := r.Write(proxyStream); err != nil {
		s.logger.LogErrorMessage().Err(err).Msgf("Failed to write request to proxy stream for client %s", tunnelID)
		return
	}

	resp, err := http.ReadResponse(bufio.NewReader(proxyStream), r)
	if err != nil {
		http.Error(w, "Error reading response from client service", http.StatusBadGateway)
		s.logger.LogErrorMessage().Err(err).Msgf("Failed to read response from proxy stream for client %s", tunnelID)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
