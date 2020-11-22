package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/evorts/shadowdp/config"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/js"
	"github.com/go-rod/rod/lib/launcher"
	"net/http"
	"strings"
	"time"
)

const jsInjection = `
	// only execute idle callback when supported
	if ('requestIdleCallback' in window) {
		window.requestIdleCallback(function (e) {
			document.body.classList.add("page-completed");
		});
	}
`

type manager struct {
	RemoteEnabled bool
	RemoteAddress string
	Browser       *rod.Browser
	Launcher      *launcher.Launcher
}

type IManager interface {
	Initiate() IManager
	GetBrowser() *rod.Browser
	GetLauncher() *launcher.Launcher
}

func NewBrowser(remoteEnabled bool, remoteAddress string) IManager {
	return &manager{
		RemoteEnabled: remoteEnabled,
		RemoteAddress: remoteAddress,
	}
}

func (m *manager) GetBrowser() *rod.Browser {
	return m.Browser
}

func (m *manager) GetLauncher() *launcher.Launcher {
	return m.Launcher
}

func (m *manager) Initiate() IManager {
	m.Browser = rod.New()
	if m.RemoteEnabled {
		// To launch remote browsers, you need a remote launcher service,
		// Rod provides a docker image for beginners, make sure have started:
		// docker run -p 9222:9222 rodorg/rod
		//
		// For more information, check the doc of launcher.RemoteLauncher
		m.Launcher = launcher.MustNewRemote(m.RemoteAddress)
		// Manipulate flags
		// l.Set("any-flag").Delete("any-flag")
		fmt.Println(m.Launcher.MustLaunch())
		m.Browser = m.Browser.Client(m.Launcher.Client())
	}
	m.Browser = m.Browser.MustConnect().MustIncognito().MustIgnoreCertErrors(true)
	// Even you forget to close, rod will close it after main process ends.
	// defer m.Browser.MustClose()
	return m
}

func goRodRender(w http.ResponseWriter, r *http.Request) {
	cfg := r.Context().Value("cfg").(config.IManager)
	cdp := r.Context().Value("cdp").(IManager)
	mapping := cfg.GetMapByHost(r.Host)
	if mapping == nil {
		_, _ = fmt.Fprint(w, "")
		return
	}
	// browser.Logger(rod.DefaultLogger).Trace(true)
	page := cdp.GetBrowser().MustPage("")
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
	// Custom function to add script tag with its content to body
	addScriptTagToBody := func(p *rod.Page, value string) error {
		var addScriptHelper = &js.Function{
			Name:         "addScriptTagToBody",
			Definition:   `function(e,t,n){if(!document.getElementById(e))return new Promise((i,o)=>{var s=document.createElement("script");t?(s.src=t,s.onload=i):(s.type="text/javascript",s.text=n,i()),s.id=e,s.onerror=o,document.body.appendChild(s)})}`,
			Dependencies: []*js.Function{},
		}
		hash := md5.Sum([]byte(value))
		id := hex.EncodeToString(hash[:])
		script := rod.EvalHelper(addScriptHelper, id, "", value)
		_, err := p.Evaluate(script.ByPromise())
		return err
	}
	headElement := page.MustElement("head")
	// prevent execution of tracking such as google analytics, gtm, or facebook
	// let's start by scanning the head section
	for _, s := range headElement.MustElements("script") {
		if strings.Contains(s.MustHTML(), "googletagmanager.com") {
			s.MustRemove()
		}
	}
	bodyElement := page.MustElement("body")
	bodyElement.MustElement("noscript").MustRemove()
	err = addScriptTagToBody(page, jsInjection)
	err = page.AddScriptTag("", jsInjection) // this one adding script tag on head section
	if err != nil {
		_, _ = fmt.Fprint(w, "")
		return
	}

	wait()
	page.MustWaitIdle().MustElement("body.page-completed")
	htmlRootElement, err2 := bodyElement.Parent()
	if err2 != nil {
		_, _ = fmt.Fprint(w, "")
		return
	}
	htmlResult := htmlRootElement.MustHTML()
	//_ = ioutil.WriteFile("rendered.html", []byte(htmlResult), os.ModePerm)
	_, _ = fmt.Fprintln(w, htmlResult)
}
