package press

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spcameron/scribe"
	"go.yaml.in/yaml/v3"
)

var fence = []byte("---")

var (
	ErrMissingTitle              = errors.New("frontmatter is missing title")
	ErrMissingSlug               = errors.New("frontmatter is missing slug")
	ErrMissingDate               = errors.New("frontmatter is missing date")
	ErrInvalidDate               = errors.New("frontmatter contains an invalid date format")
	ErrInvalidFrontMatter        = errors.New("frontmatter is malformed")
	ErrEmptyFile                 = errors.New("empty file")
	ErrMissingOpeningFence       = errors.New("missing frontmatter opening fence")
	ErrMissingClosingFence       = errors.New("missing frontmatter closing fence")
	ErrOpeningFenceNotTerminated = errors.New("opening fence missing terminating newline")
)

type Post struct {
	SourcePath string
	SourceDir  string

	Slug  string
	URL   string
	Title string
	Date  time.Time

	Body HTML
}

type frontMatter struct {
	title string
	slug  string
	date  time.Time
}

type HTML interface {
	Write(io.Writer) error
}

func discoverPosts(contentRoot string) ([]string, error) {
	if err := requireDir(contentRoot); err != nil {
		return nil, err
	}

	postsDir := filepath.Join(contentRoot, "posts")

	if err := requireDir(postsDir); err != nil {
		return nil, err
	}

	var candidates []string

	if walkErr := filepath.WalkDir(postsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}

		name := d.Name()

		if d.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		if strings.EqualFold(name, "index.md") {
			if filepath.Clean(filepath.Dir(path)) == filepath.Clean(postsDir) {
				return nil
			}
			candidates = append(candidates, path)
		}

		return nil

	}); walkErr != nil {
		return nil, walkErr
	}

	sort.Strings(candidates)
	return candidates, nil
}

func parsePosts(paths []string) ([]Post, error) {
	var posts []Post
	for _, s := range paths {
		p, err := parsePost(s)
		if err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}

	return posts, nil
}

func parsePost(path string) (Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Post{}, err
	}

	fmBytes, mdBytes, err := splitPost(data)
	if err != nil {
		return Post{}, err
	}

	fm, err := decodeFrontMatter(fmBytes)
	if err != nil {
		return Post{}, err
	}

	md, err := scribe.Compile(string(mdBytes))
	if err != nil {
		return Post{}, err
	}

	clean := filepath.Clean(path)

	post := Post{
		SourcePath: clean,
		SourceDir:  filepath.Dir(clean),

		Slug:  fm.slug,
		Title: fm.title,
		Date:  fm.date,

		Body: md,
	}

	return post, nil
}

func splitPost(src []byte) (fmBytes, mdBytes []byte, err error) {
	if len(src) == 0 {
		return nil, nil, ErrEmptyFile
	}

	i := bytes.IndexByte(src, '\n')
	if i == -1 {
		return nil, nil, ErrOpeningFenceNotTerminated
	}

	if !bytes.Equal(src[:i], fence) {
		return nil, nil, ErrMissingOpeningFence
	}

	openEnd := i + 1

	for pos := openEnd; pos < len(src); {
		rel := bytes.IndexByte(src[pos:], '\n')

		var line []byte
		var nextPos int
		if rel == -1 {
			line = src[pos:]
			nextPos = len(src)
		} else {
			j := pos + rel
			line = src[pos:j]
			nextPos = j + 1
		}

		if bytes.Equal(line, fence) {
			fmBytes = src[openEnd:pos]
			mdBytes = src[nextPos:]
			return fmBytes, mdBytes, nil
		}

		pos = nextPos
	}

	return nil, nil, ErrMissingClosingFence
}

func decodeFrontMatter(data []byte) (frontMatter, error) {
	var raw struct {
		Title string `yaml:"title"`
		Slug  string `yaml:"slug"`
		Date  string `yaml:"date"`
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	if err := dec.Decode(&raw); err != nil {
		return frontMatter{}, fmt.Errorf("%w: %v", ErrInvalidFrontMatter, err)
	}

	if strings.TrimSpace(raw.Title) == "" {
		return frontMatter{}, ErrMissingTitle
	}
	if strings.TrimSpace(raw.Slug) == "" {
		return frontMatter{}, ErrMissingSlug
	}
	if strings.TrimSpace(raw.Date) == "" {
		return frontMatter{}, ErrMissingDate
	}

	t, err := time.Parse("2006-01-02", raw.Date)
	if err != nil {
		return frontMatter{}, fmt.Errorf("decode: %w (expected YYYY-MM-DD): %q", ErrInvalidDate, raw.Date)
	}

	fm := frontMatter{
		title: raw.Title,
		slug:  raw.Slug,
		date:  t,
	}

	return fm, nil
}

func validatePosts(posts []Post) error {
	seen := make(map[string]struct{}, len(posts))
	for _, p := range posts {
		s := strings.TrimSpace(p.Slug)
		if s == "" {
			return ErrMissingSlug
		}

		if _, ok := seen[s]; ok {
			return fmt.Errorf("validate: duplicate slug: %q", s)
		}

		seen[s] = struct{}{}
	}

	return nil
}

func sortPosts(posts []Post) {
	sort.Slice(posts, func(i, j int) bool {
		if !posts[i].Date.Equal(posts[j].Date) {
			return posts[i].Date.After(posts[j].Date)
		}
		return posts[i].Title < posts[j].Title
	})
}

func requireDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", path)
	}

	return nil
}

func shouldSkipDir(name string) bool {
	if name == "" {
		return false
	}

	if strings.HasPrefix(name, ".") {
		return true
	}

	switch name {
	case "node_modules", "vendor", "tmp", "dist", "build":
		return true
	default:
		return false
	}
}
