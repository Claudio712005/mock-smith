package scenario

// ForceStatus devolve um Scenario que força um status HTTP nas requisições cujo
// endpoint documenta aquele status na spec (sucesso ou erro). Endpoints que não
// documentam o status caem em base (tipicamente o profile). Assim, forçar 500
// num profile happy faz só os endpoints com 500 mapeado responderem 500; os
// demais seguem o happy.
func ForceStatus(status int, base Scenario) Scenario {
	return forceStatus{status: status, base: base}
}

type forceStatus struct {
	status int
	base   Scenario
}

func (f forceStatus) Resolve(ctx *RequestContext) Result {
	ep := ctx.Endpoint

	if ep.SuccessResponse != nil && ep.SuccessResponse.StatusCode == f.status {
		return Result{Kind: KindSuccess}
	}

	for _, e := range ep.ErrorResponses {
		if e.StatusCode == f.status {
			return Result{Kind: KindServerError, Status: f.status}
		}
	}

	return f.base.Resolve(ctx)
}
