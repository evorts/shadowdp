package main

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"net/http"
	"time"
)

func chromeDpRender(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("request received with path: %s\n", r.URL.Path)
	ctx, cancel := chromedp.NewContext(
		context.Background(),
		chromedp.WithBrowserOption(chromedp.WithDialTimeout(30*time.Second)),
		//chromedp.WithDebugf(log.Printf),
	)
	defer cancel()
	var result string
	if err := chromedp.Run(ctx,
		chromeDpTask(fmt.Sprintf("%s%s", "https://www.ruangmom.com", r.URL.Path), &result),
	); err != nil {
		_, _ = fmt.Fprintln(w, "")
		return
	}
	//_ = ioutil.WriteFile("rendered.html", []byte(result), os.ModePerm)
	_, _ = fmt.Fprintln(w, result)
}

func enableLifeCycleEvents() chromedp.ActionFunc {
	return func(ctx context.Context) error {
		err := page.Enable().Do(ctx)
		if err != nil {
			return err
		}
		err = page.SetLifecycleEventsEnabled(true).Do(ctx)
		if err != nil {
			return err
		}
		return nil
	}
}

func navigateAndWaitFor(url string, eventName string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		_, _, _, err := page.Navigate(url).Do(ctx)
		if err != nil {
			return err
		}

		return waitFor(ctx, eventName)
	}
}

func chromeDpTask(url string, result *string) chromedp.Tasks {
	return chromedp.Tasks{
		enableLifeCycleEvents(),
		navigateAndWaitFor(url, "networkIdle"),
		chromedp.ActionFunc(func(ctx context.Context) error {
			node, err := dom.GetDocument().Do(ctx)
			if err != nil {
				return err
			}
			*result, err = dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
			return err
		}),
	}
}

// waitFor blocks until eventName is received.
// Examples of events you can wait for:
//     init, DOMContentLoaded, firstPaint,
//     firstContentfulPaint, firstImagePaint,
//     firstMeaningfulPaintCandidate,
//     load, networkAlmostIdle, firstMeaningfulPaint, networkIdle
//
// This is not super reliable, I've already found incidental cases where
// networkIdle was sent before load. It's probably smart to see how
// puppeteer implements this exactly.
func waitFor(ctx context.Context, eventName string) error {
	ch := make(chan struct{})
	c, cancel := context.WithCancel(ctx)
	chromedp.ListenTarget(c, func(ev interface{}) {
		switch e := ev.(type) {
		case *page.EventLoadEventFired:
			fmt.Printf("page loaded\n")
		case *page.EventLifecycleEvent:
			fmt.Printf("lifecycle event triggered: %s\n", e.Name)
			if e.Name == eventName {
				cancel()
				close(ch)
			}
		}
	})
	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
