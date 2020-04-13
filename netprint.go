// implements the upload of file to 7eleven netprint service
// TODO implement clearing of the queue
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"os"
	"time"
)

const (
	netprint_timeout = 40
)

func main() {
	var (
		pin      string
		filename string
		debug    bool
	)

	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.StringVar(&pin, "pin", "", "set a pin for your documents")
	flag.Parse()

	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
	}
	// in case you need to have a visible execution of Chrome
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
	// add a timeout
	ctx, cancel = context.WithTimeout(ctx, netprint_timeout*time.Second)
	defer cancel()

	// the file to be uploaded
	filename = flag.Arg(0)
	if filename == "" {
		fmt.Println("Please provide a filename as argument")
		os.Exit(1)
	}

	// run the browser to update the file
	err := chromedp.Run(ctx, login(user, pass))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if pin != "" {
		err = chromedp.Run(ctx, setpin(pin))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	// finally send the file
	err = chromedp.Run(ctx, sendfile(filename))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// in case you need to set a pin
func setpin(pin string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Click(`label[for="pin-num-set-fl-0"]`),
		chromedp.SendKeys(`input[name="pin-no"]`, "1222", chromedp.NodeVisible),
	}
}

func login(user, pass string) chromedp.Tasks {

	return chromedp.Tasks{
		chromedp.Navigate("https://www.printing.ne.jp/usr/web/NPCM0010.seam"),
		chromedp.SendKeys(`input[id="NPCM0010:userIdOrMailads-txt"]`, user, chromedp.NodeVisible),
		chromedp.SendKeys(`input[id="NPCM0010:password-pwd"]`, pass, chromedp.NodeVisible),
		chromedp.Click(`#login`),

		// login is done from there
		chromedp.Click(`.mb4 > a`),

		// wait from "File button" then clicked on it
		chromedp.WaitVisible(`#pin-no`, chromedp.ByID),
	}
}
func sendfile(filename string) chromedp.Tasks {
	return chromedp.Tasks{
		// Make A4, white-black
		chromedp.Click(`label[for="yus-size-0"]`),
		chromedp.Click(`label[for="iro-cl-2"]`),

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
