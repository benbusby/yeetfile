package api

import (
	"errors"
	"fmt"
	"net/http"
	"yeetfile/cli/clilang"
	"yeetfile/cli/requests"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

// InitStripeCheckout produces a link that the user can use to check out via Stripe
func (ctx *Context) InitStripeCheckout(upgrade shared.Upgrade, quantity string) (string, error) {
	var url string
	if upgrade.IsVaultUpgrade {
		url = fmt.Sprintf("%s?vault-upgrade=%s&vault-quantity=%s",
			endpoints.StripeCheckout.Format(ctx.Server),
			upgrade.Tag,
			quantity)
	} else {
		url = fmt.Sprintf("%s?send-upgrade=%s",
			endpoints.StripeCheckout.Format(ctx.Server),
			upgrade.Tag)
	}

	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return "", err
	} else if resp.StatusCode > http.StatusBadRequest {
		return "", shared.ParseHTTPError(resp)
	}

	redirect := resp.Header.Get("Location")
	if len(redirect) == 0 {
		return "", errors.New(clilang.I18n.T("cli.api.error.missing_checkout_link"))
	}

	return redirect, nil
}

// InitBTCPayCheckout produces a link that the user can use to check out via BTCPay
func (ctx *Context) InitBTCPayCheckout(subType, quantity string) (string, error) {
	url := fmt.Sprintf("%s?type=%s&quantity=%s",
		endpoints.BTCPayCheckout.Format(ctx.Server),
		subType,
		quantity)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return "", err
	} else if resp.StatusCode > http.StatusBadRequest {
		return "", shared.ParseHTTPError(resp)
	}

	redirect := resp.Header.Get("Location")
	if len(redirect) == 0 {
		return "", errors.New(clilang.I18n.T("cli.api.error.missing_checkout_link"))
	}

	return redirect, nil
}
