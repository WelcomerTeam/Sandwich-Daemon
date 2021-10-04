package internal

import (
	"strconv"
	"time"

	"github.com/fasthttp/router"
	"github.com/rs/zerolog"
	"github.com/valyala/fasthttp"
	"golang.org/x/xerrors"
)

var (
	ErrForbidden = xerrors.New("You do not have permission to access this")

	DiscordUserMeEndpoint = "https://discord.com/api/users/@me"
)

func (sg *Sandwich) NewRestRouter() fasthttp.RequestHandler {
	r := router.New()
	r.GET("/", sg.RequireDiscordAuthentication(sg.testEndpoint))

	return r.Handler
}

// RequireDiscordAuthentication wraps a RequestHandler and
// redirects to oauth if not in session and raises Unauthorized
// if user is not permitted.
func (sg *Sandwich) RequireDiscordAuthentication(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		h(ctx)
		return
	})
}

func (sg *Sandwich) HandleRequest(ctx *fasthttp.RequestCtx) {
	start := time.Now()

	defer func() {
		var log *zerolog.Event

		processingMS := time.Since(start).Milliseconds()
		statusCode := ctx.Response.StatusCode()

		switch {
		case (statusCode >= 400 && statusCode <= 499):
			log = sg.Logger.Warn()
		case (statusCode >= 500 && statusCode <= 599):
			log = sg.Logger.Error()
		default:
			log = sg.Logger.Info()
		}

		log.Msgf("%s %s %s %d %d %dms",
			ctx.RemoteAddr(),
			ctx.Request.Header.Method(),
			ctx.Request.URI().PathOriginal(),
			statusCode,
			len(ctx.Response.Body()),
			processingMS,
		)

		ctx.Response.Header.Set("X-Elapsed", strconv.FormatInt(processingMS, MagicDecimalBase))
	}()

	fasthttp.CompressHandlerBrotliLevel(
		sg.RouterHandler,
		fasthttp.CompressBrotliDefaultCompression,
		fasthttp.CompressDefaultCompression,
	)(ctx)
}

func (sg *Sandwich) testEndpoint(ctx *fasthttp.RequestCtx) {
	ctx.WriteString("Hello!")
	ctx.Response.SetStatusCode(200)
}
