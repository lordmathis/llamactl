package llamactl

import (
	"net/http"
	"os/exec"
)

// HelpHandler godoc
// @Summary Get help for llama server
// @Description Returns the help text for the llama server command
// @Tags server
// #Produces text/plain
// @Success 200 {string} string "Help text"
// @Failure 500 {string} string "Internal Server Error"
// @Router /server/help [get]
func HelpHandler(w http.ResponseWriter, r *http.Request) {
	helpCmd := exec.Command("llama-server", "--help")
	output, err := helpCmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Failed to get help: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}

// VersionHandler godoc
// @Summary Get version of llama server
// @Description Returns the version of the llama server command
// @Tags server
// #Produces text/plain
// @Success 200 {string} string "Version information"
// @Failure 500 {string} string "Internal Server Error"
// @Router /server/version [get]
func VersionHandler(w http.ResponseWriter, r *http.Request) {
	versionCmd := exec.Command("llama-server", "--version")
	output, err := versionCmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Failed to get version: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}

// ListDevicesHandler godoc
// @Summary List available devices for llama server
// @Description Returns a list of available devices for the llama server
// @Tags server
// #Produces text/plain
// @Success 200 {string} string "List of devices"
// @Failure 500 {string} string "Internal Server Error"
// @Router /server/devices [get]
func ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
	listCmd := exec.Command("llama-server", "--list-devices")
	output, err := listCmd.CombinedOutput()
	if err != nil {
		http.Error(w, "Failed to list devices: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(output)
}
