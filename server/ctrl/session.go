package ctrl

import (
	"errors"
	"github.com/mickael-kerjean/mux"
	. "github.com/mickael-kerjean/nuage/server/common"
	"github.com/mickael-kerjean/nuage/server/model"
	"net/http"
	"time"
)

func SessionIsValid(ctx App, res http.ResponseWriter, req *http.Request) {
	if ctx.Backend == nil {
		SendSuccessResult(res, false)
		return
	}
	if _, err := ctx.Backend.Ls("/"); err != nil {
		SendSuccessResult(res, false)
		return
	}
	home, _ := model.GetHome(ctx.Backend)
	if home == "" {
		SendSuccessResult(res, true)
		return
	}
	SendSuccessResult(res, true)
}

func SessionAuthenticate(ctx App, res http.ResponseWriter, req *http.Request) {
	ctx.Body["timestamp"] = time.Now().String()
	backend, err := model.NewBackend(&ctx, ctx.Body)
	if err != nil {
		SendErrorResult(res, err)
		return
	}

	if obj, ok := backend.(interface {
		OAuthToken(*map[string]string) error
	}); ok {
		err := obj.OAuthToken(&ctx.Body)
		if err != nil {
			SendErrorResult(res, NewError("Can't authenticate (OAuth error)", 401))
			return
		}
		backend, err = model.NewBackend(&ctx, ctx.Body)
		if err != nil {
			SendErrorResult(res, NewError("Can't authenticate", 401))
			return
		}
	}

	home, err := model.GetHome(backend)
	if err != nil {
		SendErrorResult(res, err)
		return
	}

	obfuscate, err := Encrypt(ctx.Config.General.SecretKey, ctx.Body)
	if err != nil {
		SendErrorResult(res, NewError(err.Error(), 500))
		return
	}
	cookie := http.Cookie{
		Name:     COOKIE_NAME,
		Value:    obfuscate,
		MaxAge:   60 * 60 * 24 * 30,
		Path:     COOKIE_PATH,
		HttpOnly: true,
	}
	http.SetCookie(res, &cookie)

	if home == "" {
		SendSuccessResult(res, nil)
	} else {
		SendSuccessResult(res, home)
	}
}

func SessionLogout(ctx App, res http.ResponseWriter, req *http.Request) {
	cookie := http.Cookie{
		Name:   COOKIE_NAME,
		Value:  "",
		Path:   COOKIE_PATH,
		MaxAge: -1,
	}
	if ctx.Backend != nil {
		if obj, ok := ctx.Backend.(interface{ Close() error }); ok {
			go obj.Close()
		}
	}

	http.SetCookie(res, &cookie)
	SendSuccessResult(res, nil)
}

func SessionOAuthBackend(ctx App, res http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	a := map[string]string{
		"type": vars["service"],
	}
	b, err := model.NewBackend(&ctx, a)
	if err != nil {
		SendErrorResult(res, err)
		return
	}
	obj, ok := b.(interface{ OAuthURL() string })
	if ok == false {
		SendErrorResult(res, errors.New("No backend authentication"))
		return
	}
	SendSuccessResult(res, obj.OAuthURL())
}
