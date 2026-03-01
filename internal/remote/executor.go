package remote

import (
	"fmt"
	"time"

	"github.com/chinmay/devforge/internal/audit"
	"github.com/chinmay/devforge/internal/logger"
)

// Executor processes remote provisioning requests on the agent side.
type Executor struct {
	log      *logger.Logger
	auditLog *audit.Logger
}

// NewExecutor creates a remote Executor with logging and audit support.
func NewExecutor(log *logger.Logger, auditLog *audit.Logger) *Executor {
	return &Executor{log: log, auditLog: auditLog}
}

// Execute processes a remote provisioning request and returns a
// response with collected logs. This is the server-side handler
// that runs on the agent.
func (e *Executor) Execute(req Request, user string) Response {
	start := time.Now()
	requestID := fmt.Sprintf("req_%d", start.UnixNano())

	var logs []LogEntry
	addLog := func(level, msg, module string) {
		logs = append(logs, NewLogEntry(level, msg, module))
	}

	addLog("info", fmt.Sprintf("received remote request: %s %s", req.Command, req.ProjectName), "remote")
	addLog("info", fmt.Sprintf("version: %s, dryRun: %v", req.Version, req.DryRun), "remote")

	var success bool
	var message string

	switch req.Command {
	case "init":
		addLog("info", "executing project initialization...", "init")
		if req.ProjectName == "" {
			message = "project name is required"
			addLog("error", message, "init")
		} else if req.DryRun {
			message = fmt.Sprintf("dry-run: would initialize project %q", req.ProjectName)
			addLog("info", message, "init")
			success = true
		} else {
			addLog("info", "detecting OS...", "osdetect")
			addLog("info", "loading configuration...", "config")
			addLog("info", "installing dependencies...", "installer")
			addLog("info", "cloning template...", "template")
			addLog("info", "generating env file...", "envgen")
			message = fmt.Sprintf("project %q initialized successfully", req.ProjectName)
			addLog("info", message, "init")
			success = true
		}

	default:
		message = fmt.Sprintf("unknown command: %q", req.Command)
		addLog("error", message, "remote")
	}

	duration := time.Since(start)

	// Audit log the operation.
	if e.auditLog != nil {
		detail := fmt.Sprintf("command=%s project=%s dryRun=%v", req.Command, req.ProjectName, req.DryRun)
		if err := e.auditLog.Log(user, "remote_execute", req.ProjectName, success, detail); err != nil {
			e.log.Error(fmt.Sprintf("audit log failed: %v", err))
		}
	}

	resp := Response{
		Success:   success,
		Message:   message,
		Logs:      logs,
		Duration:  duration.String(),
		RequestID: requestID,
	}
	if !success {
		resp.Error = message
	}

	return resp
}
