package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/logtrace"
)

// SendJsonRsp sends a JSON response with the given status code and message.
// If location is provided and status code is http.StatusCreated (201),
// sets the Location header. Handles both pre-marshaled JSON and structs.
func SendJsonRsp(ctx context.Context, w http.ResponseWriter, statusCode int, msg any, location ...string) {
	var msgJson []byte
	if jsonStr, ok := msg.(string); ok {
		b := []byte(jsonStr)
		if json.Valid(b) {
			msgJson = b
		}
	} else if jsonStr, ok := msg.([]byte); ok {
		if json.Valid(jsonStr) {
			msgJson = jsonStr
		}
	} else {
		var err error
		msgJson, err = json.Marshal(msg)
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("unable to marshal json")
			ErrApplicationError("Id: " + logtrace.RequestIdFromContext(ctx)).Send(w)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if statusCode == http.StatusCreated && len(location) > 0 {
		w.Header().Set("Location", location[0])
	}
	w.WriteHeader(statusCode)
	w.Write(msgJson)
}
