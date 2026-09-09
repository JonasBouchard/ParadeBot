package main

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/mxschmitt/playwright-go"
)

var errSaveUnverified = errors.New("parade save not verified; check RMC before retrying")

type paradeState struct {
	status  string
	comment string
	updated string
}

// These fields and the new_status response come from RMC's update_status page.
func readParadeState(page playwright.Page) (paradeState, error) {
	var state paradeState
	var err error
	if state.status, err = page.Locator("#statusSelector").InputValue(); err != nil {
		return state, err
	}
	if state.status == "" {
		return state, fmt.Errorf("RMC did not provide a current status")
	}
	if state.comment, err = page.Locator("#currentStatusComment").InputValue(); err != nil {
		return state, err
	}
	state.updated, err = page.Locator("#currentStatusTimestamp").InnerText()
	state.updated = normalizeTimestamp(state.updated)
	return state, err
}

func normalizeTimestamp(value string) string {
	return strings.Join(strings.Fields(html.UnescapeString(value)), " ")
}

func saveAndVerifyParade(page playwright.Page, pageURL string) error {
	// Revalidate the document on readback instead of accepting a cached page.
	if err := page.SetExtraHTTPHeaders(map[string]string{"Cache-Control": "no-cache", "Pragma": "no-cache"}); err != nil {
		return err
	}
	response, err := page.Goto(pageURL)
	if err != nil {
		return fmt.Errorf("could not open parade status: %w", err)
	}
	if response == nil || !response.Ok() {
		return fmt.Errorf("RMC did not load the parade status page successfully")
	}
	expected, err := readParadeState(page)
	if err != nil {
		return fmt.Errorf("could not read parade status before saving: %w", err)
	}

	// Register before clicking so even a fast save response is captured. Never
	// repeat this POST automatically: an uncertain response may still be saved.
	saveURL := strings.TrimRight(pageURL, "/") + "/new_status"
	response, err = page.ExpectResponse(func(response playwright.Response) bool {
		return response.URL() == saveURL && response.Request().Method() == "POST"
	}, func() error {
		return page.Locator("#statusEvenBtn").Click()
	})
	if err != nil {
		return fmt.Errorf("%w: save acknowledgement was not received: %v", errSaveUnverified, err)
	}
	if !response.Ok() {
		return fmt.Errorf("%w: save returned HTTP %d", errSaveUnverified, response.Status())
	}
	var acknowledgement struct {
		LastUpdated string `json:"last_updated"`
	}
	if err := response.JSON(&acknowledgement); err != nil {
		return fmt.Errorf("%w: save returned an invalid acknowledgement", errSaveUnverified)
	}
	acknowledgedTimestamp := normalizeTimestamp(acknowledgement.LastUpdated)
	if acknowledgedTimestamp == "" || acknowledgedTimestamp == expected.updated {
		return fmt.Errorf("%w: save did not return a new update timestamp", errSaveUnverified)
	}

	setActivity("Verifying saved parade status")
	response, err = page.Reload()
	if err != nil || response == nil || !response.Ok() {
		return fmt.Errorf("%w: could not reload the saved status", errSaveUnverified)
	}
	saved, err := readParadeState(page)
	if err != nil {
		return fmt.Errorf("%w: saved status could not be read after reload", errSaveUnverified)
	}
	if saved.status != expected.status || saved.comment != expected.comment || saved.updated != acknowledgedTimestamp {
		return fmt.Errorf("%w: reloaded status, comment, or timestamp does not match the save", errSaveUnverified)
	}
	return nil
}
