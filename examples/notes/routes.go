package main

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	session "github.com/kohkimakimoto/echo-session/v5"
	"github.com/kohkimakimoto/inertia-echo/examples/notes/internal/store"
	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
)

func userID(c *echo.Context) string { return session.MustGet(c).GetString("user_id") }
func flash(c *echo.Context, message string) {
	inertia.Flash(c, map[string]any{"message": message, "type": "success"})
}
func redirect(c *echo.Context, path string) error { return c.Redirect(http.StatusSeeOther, path) }
func notFound(c *echo.Context) error {
	return inertia.Render(c, http.StatusNotFound, "Errors/NotFound", map[string]any{"status": 404})
}

func registerRoutes(e *echo.Echo, db *store.Store) {
	auth := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if _, ok := db.GetUser(userID(c)); !ok {
				return redirect(c, "/login")
			}
			return next(c)
		}
	}
	e.GET("/login", func(c *echo.Context) error {
		if _, ok := db.GetUser(userID(c)); ok {
			inertia.ClearHistory(c)
			return redirect(c, "/")
		}
		return inertia.Render(c, http.StatusOK, "Login", map[string]any{})
	})
	e.POST("/login", func(c *echo.Context) error {
		var form struct {
			Email    string `json:"email" form:"email"`
			Password string `json:"password" form:"password"`
		}
		if err := c.Bind(&form); err != nil {
			return err
		}
		user, ok := db.FindUserByEmail(strings.TrimSpace(form.Email))
		if !ok || user.Password != form.Password {
			return inertia.Render(c, http.StatusOK, "Login", map[string]any{"errors": map[string]string{"email": "Invalid email or password."}})
		}
		s := session.MustGet(c)
		s.Clear()
		// Clear expires the old cookie; restore a lifetime for the new login.
		s.Options().MaxAge = 86400
		s.Set("user_id", user.ID)
		if err := s.Save(); err != nil {
			return err
		}
		flash(c, "Welcome back, "+user.Name+".")
		inertia.ClearHistory(c)
		return redirect(c, "/")
	})
	e.POST("/logout", func(c *echo.Context) error {
		s := session.MustGet(c)
		s.Clear()
		if err := s.Save(); err != nil {
			return err
		}
		flash(c, "You have been logged out.")
		inertia.ClearHistory(c)
		return redirect(c, "/login")
	}, auth)
	e.GET("/", func(c *echo.Context) error {
		uid := userID(c)
		stats := func() (any, error) {
			// Explicit failure path demonstrates Rescue without breaking the rest of the page.
			if c.QueryParam("failStats") == "1" {
				return nil, errors.New("intentional stats demo failure")
			}
			notes, total := db.ListNotes(uid, store.Filter{})
			pinned := 0
			for _, n := range notes {
				if n.Pinned {
					pinned++
				}
			}
			return map[string]int{"note_count": total, "pinned_count": pinned, "tag_count": len(db.Tags(uid))}, nil
		}
		recent := func() (any, error) {
			notes, _ := db.ListNotes(uid, store.Filter{})
			sort.SliceStable(notes, func(i, j int) bool { return notes[i].UpdatedAt.After(notes[j].UpdatedAt) })
			if len(notes) > 5 {
				notes = notes[:5]
			}
			return notes, nil
		}
		var recentProp any = inertia.DeferWithGroup(recent, "recent")
		// Pinning from Dashboard partially reloads recent and matches existing rows by ID.
		if c.QueryParam("merge") == "1" {
			notes, _ := recent()
			recentProp = inertia.Merge(notes).MatchOn("id")
		}
		return inertia.Render(c, http.StatusOK, "Dashboard", map[string]any{"stats": inertia.DeferWithGroup(stats, "stats").Rescue(), "recent": recentProp})
	}, auth)
	e.GET("/notes", func(c *echo.Context) error {
		uid := userID(c)
		q, tag := c.QueryParam("q"), c.QueryParam("tag")
		page, _ := strconv.Atoi(c.QueryParam("page"))
		if page < 1 {
			page = 1
		}
		if page > 1000000 {
			page = 1000000
		}
		size := db.GetSettings(uid).PageSize
		return inertia.Render(c, http.StatusOK, "Notes/Index", map[string]any{
			"filters":       inertia.Always(map[string]string{"q": q, "tag": tag}),
			"availableTags": db.Tags(uid),
			"notes": inertia.Scroll(func() (inertia.ScrollResult, error) {
				notes, total := db.ListNotes(uid, store.Filter{Query: q, Tag: tag, Offset: (page - 1) * size, Limit: size})
				var prev, next any
				if page > 1 {
					prev = page - 1
				}
				if page*size < total {
					next = page + 1
				}
				return inertia.ScrollResult{Value: map[string]any{"data": notes}, Metadata: inertia.ScrollMetadata{PageName: "page", PreviousPage: prev, CurrentPage: page, NextPage: next}}, nil
			}),
		})
	}, auth)
	e.GET("/notes/new", func(c *echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Notes/Edit", map[string]any{"note": store.NoteInput{Tags: []string{}}})
	}, auth)
	e.GET("/notes/:id", func(c *echo.Context) error {
		n, ok := db.GetNote(userID(c), c.Param("id"))
		if !ok {
			return notFound(c)
		}
		return inertia.Render(c, http.StatusOK, "Notes/Show", map[string]any{"note": n})
	}, auth)
	e.GET("/notes/:id/edit", func(c *echo.Context) error {
		n, ok := db.GetNote(userID(c), c.Param("id"))
		if !ok {
			return notFound(c)
		}
		return inertia.Render(c, http.StatusOK, "Notes/Edit", map[string]any{"note": n})
	}, auth)
	saveNote := func(c *echo.Context) error {
		id := c.Param("id")
		uid := userID(c)
		if id != "" {
			if _, ok := db.GetNote(uid, id); !ok {
				return notFound(c)
			}
		}
		var input store.NoteInput
		if err := c.Bind(&input); err != nil {
			return err
		}
		input.Title = strings.TrimSpace(input.Title)
		validation := map[string]string{}
		if input.Title == "" {
			validation["title"] = "A title is required."
		}
		if len(input.Title) > 200 {
			validation["title"] = "Use at most 200 bytes for the title."
		}
		if len(input.Body) > 100000 {
			validation["body"] = "Use at most 100,000 bytes for the body."
		}
		if len(validation) > 0 {
			return inertia.Render(c, http.StatusOK, "Notes/Edit", map[string]any{"note": map[string]any{"id": id, "title": input.Title, "body": input.Body, "tags": input.Tags, "pinned": input.Pinned}, "errors": validation})
		}
		var n store.Note
		var err error
		if id == "" {
			n, err = db.CreateNote(uid, input)
		} else {
			n, err = db.UpdateNote(uid, id, input)
		}
		if err != nil {
			return err
		}
		flash(c, "Note saved.")
		return redirect(c, "/notes/"+n.ID)
	}
	e.POST("/notes", saveNote, auth)
	e.POST("/notes/:id", saveNote, auth)
	e.POST("/notes/:id/pin", func(c *echo.Context) error {
		if _, err := db.TogglePin(userID(c), c.Param("id")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return notFound(c)
			}
			return err
		}
		flash(c, "Pin updated.")
		if c.QueryParam("dashboard") == "1" {
			return redirect(c, "/?merge=1")
		}
		return redirect(c, "/notes/"+c.Param("id"))
	}, auth)
	e.POST("/notes/:id/delete", func(c *echo.Context) error {
		if err := db.DeleteNote(userID(c), c.Param("id")); err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return notFound(c)
			}
			return err
		}
		flash(c, "Note deleted.")
		return redirect(c, "/notes")
	}, auth)
	e.POST("/notes/:id/fragment", func(c *echo.Context) error {
		if _, ok := db.GetNote(userID(c), c.Param("id")); !ok {
			return notFound(c)
		}
		// The client retains the source #body fragment across this redirect.
		inertia.PreserveFragment(c)
		return redirect(c, "/settings")
	}, auth)
	e.GET("/settings", func(c *echo.Context) error {
		uid := userID(c)
		s := session.MustGet(c)
		fresh := s.GetString("settings_fresh") == "1"
		if fresh {
			s.Set("settings_fresh", "")
			if err := s.Save(); err != nil {
				return err
			}
		}
		return inertia.Render(c, http.StatusOK, "Settings", map[string]any{"settings": inertia.Once(func() (any, error) { return db.GetSettings(uid), nil }).As("user-settings-" + uid).For(10 * time.Minute).Fresh(fresh)})
	}, auth)
	e.POST("/settings", func(c *echo.Context) error {
		var input store.Settings
		if err := c.Bind(&input); err != nil {
			return err
		}
		validation := map[string]string{}
		if input.Theme != "system" && input.Theme != "light" && input.Theme != "dark" {
			validation["theme"] = "Choose system, light, or dark."
		}
		if input.PageSize < 5 || input.PageSize > 50 {
			validation["page_size"] = "Choose a page size between 5 and 50."
		}
		if len(validation) > 0 {
			return inertia.Render(c, http.StatusOK, "Settings", map[string]any{"settings": input, "errors": validation})
		}
		if err := db.SaveSettings(userID(c), input); err != nil {
			return err
		}
		s := session.MustGet(c)
		s.Set("settings_fresh", "1")
		if err := s.Save(); err != nil {
			return err
		}
		flash(c, "Settings saved.")
		return redirect(c, "/settings")
	}, auth)
	e.RouteNotFound("/*", notFound)
}
