package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/evorts/rod"
	"github.com/evorts/shadowdp/config"
	"github.com/go-rod/rod/lib/js"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

const jsInjection = `
	if ('requestIdleCallback' in window) {
		// Use requestIdleCallback to schedule work.
		window.requestIdleCallback(function (e) {
			document.body.classList.add("page-completed");
		});
	} else {
		// Do what you’d do today.
	}
`

func goRodRender(w http.ResponseWriter, r *http.Request) {
	cfg := r.Context().Value("logger").(config.IManager)
	mapping := cfg.GetMapByHost(r.Host)
	if mapping == nil {
		_, _ = fmt.Fprint(w, "")
		return
	}
	browser := rod.New().MustConnect().MustIncognito().MustIgnoreCertErrors(true)
	// Even you forget to close, rod will close it after main process ends.
	defer browser.MustClose()
	page := browser.MustPage("")
	//wait := page.WaitRequestIdle(5 * time.Second, []string{}, []string{})
	wait := page.MustWaitRequestIdle()
	err := rod.Try(func() {
		page.
			Context(r.Context()).
			Timeout(10 * time.Second).
			MustNavigate(fmt.Sprintf("%s%s", mapping.ToBaseUrl, r.URL.Path))
	})
	if errors.Is(err, context.DeadlineExceeded) {
		// in case want to handle on timeout differently
		_, _ = fmt.Fprint(w, "")
		return
	} else if err != nil {
		_, _ = fmt.Fprint(w, "")
		return
	}
	// Convert name and jsArgs to Page.Eval, the name is method name in the "lib/js/helper.js".
	addScriptTagToBody := func(page *rod.Page, value string) error {
		hash := md5.Sum([]byte(value))
		id := hex.EncodeToString(hash[:])
		var addScriptHelper = &js.Function{
			Name:         "addScriptTagToBody",
			Definition:   `function(e,t,n){if(!document.getElementById(e))return new Promise((i,o)=>{var s=document.createElement("script");t?(s.src=t,s.onload=i):(s.type="text/javascript",s.text=n,i()),s.id=e,s.onerror=o,document.body.appendChild(s)})}`,
			Dependencies: []*js.Function{},
		}

		_, err := page.Evaluate(rod.JsHelper(addScriptHelper, id, "", value).ByPromise())
		return err
	}
	err = addScriptTagToBody(page, jsInjection)
	//err = page.AddScriptTag("", jsInjection)
	if err != nil {
		_, _ = fmt.Fprint(w, "")
		return
	}
	wait()
	//bodyElement := page.MustWaitLoad().MustElement("body")
	bodyElement := page.MustWaitIdle().MustElement("body")
	htmlRootElement, err2 := bodyElement.Parent()
	if err2 != nil {
		_, _ = fmt.Fprint(w, "")
		return
	}
	htmlResult := htmlRootElement.MustHTML()
	_ = ioutil.WriteFile("rendered.html", []byte(htmlResult), os.ModePerm)
	_, _ = fmt.Fprintln(w, htmlResult)
}
