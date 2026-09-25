package backends

import (
	"fmt"
	"llamactl/pkg/validation"
	"slices"
	"strconv"
	"strings"
)

// CustomServerOptions runs a user-defined command configured as
// backends.custom.<name> in the config file. Args are literal strings,
// not a flag grammar; {port} and {model} are substituted at build time.
type CustomServerOptions struct {
	Name       string   `json:"name,omitempty"`
	Model      string   `json:"model,omitempty"`
	Host       string   `json:"host,omitempty"`
	Port       int      `json:"port,omitempty"`
	Args       []string `json:"args,omitempty"`
	HealthPath string   `json:"health_path,omitempty"`
}

func (o *CustomServerOptions) GetModel() string {
	if o == nil {
		return ""
	}
	return o.Model
}

func (o *CustomServerOptions) GetPort() int {
	if o == nil {
		return 0
	}
	return o.Port
}

func (o *CustomServerOptions) SetPort(port int) {
	if o == nil {
		return
	}
	o.Port = port
}

func (o *CustomServerOptions) GetHost() string {
	if o == nil {
		return "localhost"
	}
	return o.Host
}

func (o *CustomServerOptions) GetHealthPath() string {
	if o == nil || o.HealthPath == "" {
		return "/health"
	}
	if !strings.HasPrefix(o.HealthPath, "/") {
		return "/" + o.HealthPath
	}
	return o.HealthPath
}

func (o *CustomServerOptions) Validate() error {
	if o == nil {
		return validation.ValidationError(fmt.Errorf("custom server options cannot be nil for custom backend"))
	}

	if o.Name == "" {
		return validation.ValidationError(fmt.Errorf("custom backend requires a name referencing a backends.custom.<name> config entry"))
	}

	if o.Port < 0 || o.Port > 65535 {
		return validation.ValidationError(fmt.Errorf("invalid port range: %d", o.Port))
	}

	hasModelPlaceholder := slices.ContainsFunc(o.Args, func(arg string) bool {
		return strings.Contains(arg, "{model}")
	})
	if hasModelPlaceholder && o.Model == "" {
		return validation.ValidationError(fmt.Errorf("custom backend args use the {model} placeholder but no model is set"))
	}

	return nil
}

// BuildCommandArgs returns the instance args with {port} and {model}
// substituted. The original Args slice is never modified because options
// are persisted and rebuilt on every command construction.
func (o *CustomServerOptions) BuildCommandArgs() []string {
	if o == nil {
		return []string{}
	}

	args := make([]string, len(o.Args))
	for i, arg := range o.Args {
		arg = strings.ReplaceAll(arg, "{port}", strconv.Itoa(o.Port))
		arg = strings.ReplaceAll(arg, "{model}", o.Model)
		args[i] = arg
	}

	return args
}

func (o *CustomServerOptions) BuildDockerArgs() []string {
	return o.BuildCommandArgs()
}

// ParseCommand is unsupported: a custom backend has no flag grammar to parse.
func (o *CustomServerOptions) ParseCommand(command string) (any, error) {
	return nil, validation.ValidationError(fmt.Errorf("parse-command is not supported for custom backends"))
}
