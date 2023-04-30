package rest

import (
	"fmt"
	"net/http"
	"strconv"
)

const (
	revisionHeader = "Etag"
)

func checkRevisionIsSame(stateRevision uint64, req *http.Request) bool {
	if len(req.Header[revisionHeader]) != 1 {
		return false
	}

	providedRevision, err := strconv.ParseUint(req.Header[revisionHeader][0], 10, 64)
	return err == nil && providedRevision == stateRevision
}

func setRevisionInResponse(stateRevision uint64, res http.ResponseWriter) {
	res.Header().Add(revisionHeader, fmt.Sprintf("%d", stateRevision))
}
