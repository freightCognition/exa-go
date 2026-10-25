package exa

func init() {
	registerSchema("ErrorResponse", ErrorResponse{})
	registerSchema("AgentErrorResponse", AgentErrorResponse{})
	registerSchema("AgentError", AgentError{})
}
