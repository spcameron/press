package press

import "path/filepath"

func assignPostURLs(posts []Post) {
	for i := range posts {
		posts[i].URL = postURL(posts[i].Slug)
	}
}

func postURL(slug string) string {
	return "/blog/" + slug + "/"
}

func blogIndexOutputPath(outDir string) string {
	return filepath.Join(outDir, "blog", "index.html")
}

func postOutputPath(outDir, slug string) string {
	return filepath.Join(outDir, "blog", slug, "index.html")
}

func postMediaSourceDir(p Post) string {
	return filepath.Join(p.SourceDir, "media")
}

func postMediaOutputDir(outDir, slug string) string {
	return filepath.Join(outDir, "blog", slug, "media")
}
