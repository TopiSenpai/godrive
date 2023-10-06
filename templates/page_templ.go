// Code generated by templ@v0.2.364 DO NOT EDIT.

package templates

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import "context"
import "io"
import "bytes"

import (
	"crypto/md5"
	"fmt"
	"strings"
)

type User struct {
	ID      string
	Name    string
	Email   string
	Home    string
	IsAdmin bool
	IsUser  bool
	IsGuest bool
}

func Page(theme string, auth bool, user User) templ.Component {
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
		_, err = templBuffer.WriteString("<!doctype html>")
		if err != nil {
			return err
		}
		var var_2 = []any{theme}
		err = templ.RenderCSSItems(ctx, templBuffer, var_2...)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("<html lang=\"en\" class=\"")
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString(templ.EscapeString(templ.CSSClasses(var_2).String()))
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("\"><head><meta charset=\"utf-8\"><title>")
		if err != nil {
			return err
		}
		var_3 := `godrive`
		_, err = templBuffer.WriteString(var_3)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</title><meta name=\"description\" content=\"godrive is a simple file sharing service\"><link rel=\"stylesheet\" type=\"text/css\" href=\"/assets/css/main.css\"><link rel=\"icon\" href=\"/assets/favicon.png\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><meta name=\"theme-color\" content=\"#1f2228\"></head><body>")
		if err != nil {
			return err
		}
		err = Header(auth, user).Render(ctx, templBuffer)
		if err != nil {
			return err
		}
		err = var_1.Render(ctx, templBuffer)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("<script src=\"/assets/js/htmx.min.js\">")
		if err != nil {
			return err
		}
		var_4 := ``
		_, err = templBuffer.WriteString(var_4)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</script><script src=\"/assets/js/main.js\">")
		if err != nil {
			return err
		}
		var_5 := ``
		_, err = templBuffer.WriteString(var_5)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</script></body></html>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}

func gravatarURL(email string) string {
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=retro", md5.Sum([]byte(strings.ToLower(email))), 80)
}

func Header(auth bool, user User) templ.Component {
	return templ.ComponentFunc(func(ctx context.Context, w io.Writer) (err error) {
		templBuffer, templIsBuffer := w.(*bytes.Buffer)
		if !templIsBuffer {
			templBuffer = templ.GetBuffer()
			defer templ.ReleaseBuffer(templBuffer)
		}
		ctx = templ.InitializeContext(ctx)
		var_6 := templ.GetChildren(ctx)
		if var_6 == nil {
			var_6 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		_, err = templBuffer.WriteString("<header><div><a title=\"godrive\" id=\"title\" href=\"/\">")
		if err != nil {
			return err
		}
		var_7 := `godrive`
		_, err = templBuffer.WriteString(var_7)
		if err != nil {
			return err
		}
		_, err = templBuffer.WriteString("</a><a title=\"GitHub\" id=\"github\" class=\"icon-btn\" href=\"https://github.com/topi314/godrive\" target=\"_blank\"></a><input id=\"theme\" class=\"toggle\" type=\"checkbox\"><label title=\"Theme\" class=\"icon-btn\" for=\"theme\"></label></div><div>")
		if err != nil {
			return err
		}
		if auth {
			if user.Name != "guest" {
				_, err = templBuffer.WriteString("<input id=\"user-menu\" type=\"checkbox\" autocomplete=\"off\"> <label title=\"")
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString(templ.EscapeString(user.Name))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("\" for=\"user-menu\"><img src=\"")
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString(templ.EscapeString(gravatarURL(user.Email)))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("\" alt=\"")
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString(templ.EscapeString(user.Name + "image"))
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("\"></label> <nav>")
				if err != nil {
					return err
				}
				if user.IsAdmin {
					_, err = templBuffer.WriteString("<a href=\"/settings\">")
					if err != nil {
						return err
					}
					var_8 := `Settings`
					_, err = templBuffer.WriteString(var_8)
					if err != nil {
						return err
					}
					_, err = templBuffer.WriteString("</a>")
					if err != nil {
						return err
					}
				}
				_, err = templBuffer.WriteString("<a href=\"/logout\">")
				if err != nil {
					return err
				}
				var_9 := `Logout`
				_, err = templBuffer.WriteString(var_9)
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("</a></nav>")
				if err != nil {
					return err
				}
			} else {
				_, err = templBuffer.WriteString("<a id=\"login\" class=\"btn primary\" href=\"/login\">")
				if err != nil {
					return err
				}
				var_10 := `Login`
				_, err = templBuffer.WriteString(var_10)
				if err != nil {
					return err
				}
				_, err = templBuffer.WriteString("</a>")
				if err != nil {
					return err
				}
			}
		}
		_, err = templBuffer.WriteString("</div></header>")
		if err != nil {
			return err
		}
		if !templIsBuffer {
			_, err = templBuffer.WriteTo(w)
		}
		return err
	})
}