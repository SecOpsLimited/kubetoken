package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/duosecurity/duo_api_golang"
)

// DuoAuth inserts a Duo auth middleware before next.
func DuoAuth(next http.Handler, duoIKey, duoSKey, duoAPIHost string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		staffid, _, ok := req.BasicAuth()
		if !ok {
			if !ok {
				http.Error(w, "Forbidden", 403)
				return
			}
		}
		if err := duoAuth(staffid, duoIKey, duoSKey, duoAPIHost); err != nil {
			http.Error(w, err.Error(), 403)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func duoAuth(staffid, duoIKey, duoSKey, duoAPIHost string) error {
	const userAgent = "kubetoken/1.0"
	duo := duoapi.NewDuoApi(duoIKey, duoSKey, duoAPIHost, userAgent)
	params := make(url.Values)
	params.Add("username", staffid)
	params.Add("factor", "auto")
	params.Add("device", "auto")
	resp, body, err := duo.SignedCall("POST", "/auth/v2/auth", params, duoapi.UseTimeout)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("expected 200, got %v: %s", resp.Status, body)
	}

	// you'd think at this point that the request was approved, oh no, not so grasshopper.
	// duo returns a 200 with a json body which contains a result key which has the words
	// "allow", "deny", so we must inspect that
	var result struct {
		Stat     string `json:"stat"`
		Response struct {
			Result  string `json:"result"`
			Status  string `json:"status"`
			Message string `json:"status_msg"`
		} `json:"response"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}
	if result.Stat != "OK" {
		return fmt.Errorf("request failed: %s", body)
	}
	if result.Response.Result != "allow" {
		return fmt.Errorf("request denied: %s: %s", result.Response.Status, result.Response.Message)
	}
	return nil
}
