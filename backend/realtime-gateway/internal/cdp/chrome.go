package cdp

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/chromedp"
)

const debugPort = "ws://localhost:9222"
const connectTimeout = 2 * time.Second

type ChromeController struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	ctx         context.Context
	cancel      context.CancelFunc
}

func NewChromeController() (*ChromeController, error) {
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), debugPort)

	ctx, cancel := chromedp.NewContext(allocCtx)

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, connectTimeout)
	defer timeoutCancel()

	if err := chromedp.Run(timeoutCtx); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("cdp: chrome not available on %s: %w", debugPort, err)
	}

	return &ChromeController{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

func (c *ChromeController) Click(selector string) error {
	return chromedp.Run(c.ctx, chromedp.Click(selector, chromedp.ByQuery))
}

func (c *ChromeController) Type(selector, text string) error {
	return chromedp.Run(c.ctx,
		chromedp.Click(selector, chromedp.ByQuery),
		chromedp.SendKeys(selector, text, chromedp.ByQuery),
	)
}

func (c *ChromeController) Navigate(url string) error {
	return chromedp.Run(c.ctx, chromedp.Navigate(url))
}

func (c *ChromeController) Scroll(deltaX, deltaY int) error {
	js := fmt.Sprintf("window.scrollBy(%d, %d)", deltaX, deltaY)
	return chromedp.Run(c.ctx, chromedp.Evaluate(js, nil))
}

func (c *ChromeController) EvaluateJS(js string) (string, error) {
	var result string
	err := chromedp.Run(c.ctx, chromedp.Evaluate(js, &result))
	return result, err
}

func (c *ChromeController) Screenshot() ([]byte, error) {
	var buf []byte
	if err := chromedp.Run(c.ctx, chromedp.FullScreenshot(&buf, 90)); err != nil {
		return nil, fmt.Errorf("cdp: screenshot failed: %w", err)
	}
	return buf, nil
}

func (c *ChromeController) Close() {
	c.cancel()
	c.allocCancel()
}
