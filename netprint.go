package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"io/ioutil"
	"math"
	"os"
	"time"
)

var (
	msgChann = make(chan cdproto.Message)
)

func main() {
	var (
		sz       string
		filename string
		debug    bool
	)

	flag.BoolVar(&debug,"debug", false, "debug mode")
	flag.Parse()

	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
	}
	if !debug {
		opts = append(opts, chromedp.Headless)
	}

	if os.Getenv("https_proxy") != "" {
		opts = append(opts, chromedp.ProxyServer(os.Getenv("https_proxy")))
	}
	actx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()
	user := os.Getenv("NETPRINT_USER")
	if user == "" {
		fmt.Printf("Input an user :")
		user, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
	pass := os.Getenv("NETPRINT_PASS")
	if pass == "" {
		fmt.Printf("Input a pass :")
		pass, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}

	ctx, cancel := chromedp.NewContext(actx)
	ctx, cancel = context.WithTimeout(ctx, 40*time.Second)
	defer cancel()

	// the file to be uploaded
	filename = flag.Arg(0)
	if filename == "" {
		fmt.Println("Please provide a filename as argument")
		os.Exit(1)
	}

	// run the browser to update the file
	err := chromedp.Run(ctx, login(user, pass, filename, &sz))
	if err != nil {
		fmt.Println(err, sz)
		os.Exit(1)
	}
}

func takeScreenshot(filename string) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// get layout metrics
		_, _, contentSize, err := page.GetLayoutMetrics().Do(ctx)
		if err != nil {
			fmt.Println("Im here")
			return err
		}

		width, height := int64(math.Ceil(contentSize.Width)), int64(math.Ceil(contentSize.Height))

		// force viewport emulation
		err = emulation.SetDeviceMetricsOverride(width, height, 1, false).
			WithScreenOrientation(&emulation.ScreenOrientation{
				Type:  emulation.OrientationTypePortraitPrimary,
				Angle: 0,
			}).Do(ctx)
		if err != nil {
			fmt.Println("Here 2")
			return err
		}

		// capture screenshot
		buf := make([]byte, 500000)
		buf, err = page.CaptureScreenshot().
			WithQuality(90).
			WithClip(&page.Viewport{
				X:      contentSize.X,
				Y:      contentSize.Y,
				Width:  contentSize.Width,
				Height: contentSize.Height,
				Scale:  1,
			}).Do(ctx)
		ioutil.WriteFile(filename, buf, 0644)
		if err != nil {
			fmt.Println("Here 3")
			return err
		}
		return nil
	})
}

func login(user, pass, filename string, sz *string) chromedp.Tasks {

	return chromedp.Tasks{
		chromedp.Navigate("https://www.printing.ne.jp/usr/web/NPCM0010.seam"),
		chromedp.SendKeys(`input[id="NPCM0010:userIdOrMailads-txt"]`, user, chromedp.NodeVisible),
		chromedp.SendKeys(`input[id="NPCM0010:password-pwd"]`, pass, chromedp.NodeVisible),
		chromedp.Click(`#login`),

		// login is done from there
		//chromedp.WaitVisible(`.mb4 > a`),
		chromedp.Click(`.mb4 > a`),

		// wait from "File button" then clicked on it
		chromedp.WaitVisible(`#pin-no`, chromedp.ByID),

		// Make A4, white-black
		chromedp.Click(`label[for="yus-size-0"]`),
		chromedp.Click(`label[for="iro-cl-2"]`),

		// in case you need to set a pin
		//chromedp.Click(`label[for="pin-num-set-fl-0"]`),
		//chromedp.SendKeys(`input[name="pin-no"]`, "1222", chromedp.NodeVisible),

		// send the filename to the field after making it "visible" otherwise it wouldn't accept to be changed
		chromedp.ActionFunc(func(ctx context.Context) error {
			_, exp, err := runtime.Evaluate(`document.querySelector("#upload-document").style.display='';`).Do(ctx)

			if err != nil {
				return err
			}
			if exp != nil {
				return exp
			}
			return nil
		}),
		//chromedp.SendKeys(`input[name="upload-document"]`, filename, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.SetUploadFiles(`input[name="upload-document"`, []string{filename}, chromedp.BySearch, chromedp.NodeVisible),
		chromedp.Click(`button[id="update-btn"]`, chromedp.NodeNotVisible, chromedp.BySearch),
		// Wait until the page is back, then logoff
		chromedp.WaitReady(`#logout-btn`, chromedp.ByID),
	}
}
