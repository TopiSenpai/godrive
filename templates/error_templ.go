// Code generated by templ@v0.2.364 DO NOT EDIT.

package templates

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"strconv"
)

type ErrorVars struct {
	Error     error
	Status    int
	Path      string
	RequestID string
}

func Error(theme string, auth bool, user User, vars ErrorVars) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_1 := templ.GetChildren(ctx)
		if var_1 == nil {
			var_1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_2 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			_, err = templBuffer.WriteString("<style>")
			if err != nil {
				return err
			}
			var_3 := `
			main {
				background-color: var(--background);
				color: var(--text-primary);
			}
		`
			_, err = templBuffer.WriteString(var_3)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</style> <main><div class=\"error\"><h1>")
			if err != nil {
				return err
			}
			var_4 := `Oops!`
			_, err = templBuffer.WriteString(var_4)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h1><h2>")
			if err != nil {
				return err
			}
			var_5 := `Something went wrong:`
			_, err = templBuffer.WriteString(var_5)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h2><div class=\"error-details\"><p>")
			if err != nil {
				return err
			}
			var_6 := `Message: `
			_, err = templBuffer.WriteString(var_6)
			if err != nil {
				return err
			}
			var var_7 string = vars.Error.Error()
			_, err = templBuffer.WriteString(templ.EscapeString(var_7))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p><p>")
			if err != nil {
				return err
			}
			var_8 := `Status: `
			_, err = templBuffer.WriteString(var_8)
			if err != nil {
				return err
			}
			var var_9 string = strconv.Itoa(vars.Status)
			_, err = templBuffer.WriteString(templ.EscapeString(var_9))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p><p>")
			if err != nil {
				return err
			}
			var_10 := `Path: `
			_, err = templBuffer.WriteString(var_10)
			if err != nil {
				return err
			}
			var var_11 string = vars.Path
			_, err = templBuffer.WriteString(templ.EscapeString(var_11))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p><p>")
			if err != nil {
				return err
			}
			var_12 := `Request ID: `
			_, err = templBuffer.WriteString(var_12)
			if err != nil {
				return err
			}
			var var_13 string = vars.RequestID
			_, err = templBuffer.WriteString(templ.EscapeString(var_13))
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p></div><h3>")
			if err != nil {
				return err
			}
			var_14 := `Try again later.`
			_, err = templBuffer.WriteString(var_14)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString(" <br> ")
			if err != nil {
				return err
			}
			var_15 := `Or create an issue on `
			_, err = templBuffer.WriteString(var_15)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("<a href=\"https://github.com/topi314/gobin/issues/new\">")
			if err != nil {
				return err
			}
			var_16 := `GitHub`
			_, err = templBuffer.WriteString(var_16)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</a></h3></div></main>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = Page(theme, auth, user).Render(templ.WithChildren(ctx, var_2), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func NotFound(theme string, auth bool, user User) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_17 := templ.GetChildren(ctx)
		if var_17 == nil {
			var_17 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var_18 := templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
			templBuffer, templIsBuffer := w.(*bytes.Buffer)
			if !templIsBuffer {
				templBuffer = templ.GetBuffer()
				defer templ.ReleaseBuffer(templBuffer)
			}
			_, err = templBuffer.WriteString("<main><div><h1>")
			if err != nil {
				return err
			}
			var_19 := `404`
			_, err = templBuffer.WriteString(var_19)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</h1><p>")
			if err != nil {
				return err
			}
			var_20 := `Page not found`
			_, err = templBuffer.WriteString(var_20)
			if err != nil {
				return err
			}
			_, err = templBuffer.WriteString("</p></div></main>")
			if err != nil {
				return err
			}
			if !templIsBuffer {
				_, err = io.Copy(w, templBuffer)
			}
			return err
		})
		err = Page(theme, auth, user).Render(templ.WithChildren(ctx, var_18), templBuffer)
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func ErrorRs(errr error) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_21 := templ.GetChildren(ctx)
		if var_21 == nil {
			var_21 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<span>")
		if err != nil {
			return err
		}
		var var_22 string = errr.Error()
		_, err = templBuffer.WriteString(templ.EscapeString(var_22))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</span>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}