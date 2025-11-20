package main

import (
	_ "embed"
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"image"
	"image/draw"
	"image/jpeg"
	"golang.org/x/image/bmp"

	"github.com/gen2brain/go-fitz"
	"github.com/briandowns/spinner"

	"thumbnailer/vips"
)

const (
	// file smaller than 10kb is served as is
	thumbMinSize = 10240
)

var (
	//go:embed version.txt
	version string

	//go:embed script.js
	jsContent string

	//go:embed style.css
	cssContent string

	//go:embed thumbnailer.ico
	favicon []byte

	cfg Config
	mu sync.Mutex
	fileInfos []FileInfo
	fileFormats = []string{
		"jpg", "jpeg", "png", "gif", "bmp", "webp", "heic", "avif", "svg", "tiff", "jp2", "jxl", "pdf", "epub", "mobi", "azw3",
	}
	vipsJpegO = &vips.JpegsaveBufferOptions{ Q: 85, }
)

type Config struct {
	cd		bool
	flat    bool
	ip      string
	lsd     bool
	open	bool
	port    uint
	sa      bool
	sd      bool
	sh      bool
	version bool
	verbose bool
	width   uint
	fit		bool
}

type ContextData struct {
	Id int `json:"id"`
}

type FileInfo struct {
	isDir   bool
	isImage bool /* refer to fileFormats */
	ID      int
	cPage   int
	modTime	int64
	Path    string
	Name    string
}

type EpubItem struct {
	Href      string `xml:"href,attr"`
	ID        string `xml:"id,attr"`
	MediaType string `xml:"media-type,attr"`
}

type EpubManifest struct {
	Items []EpubItem `xml:"item"`
}

type EpubMetadata struct {
	Meta []struct {
		Name    string `xml:"name,attr"`
		Content string `xml:"content,attr"`
	} `xml:"meta"`
}

type EpubOPF struct {
	Metadata EpubMetadata `xml:"metadata"`
	Manifest EpubManifest `xml:"manifest"`
}

type EpubOPFSimple struct {
	XMLName  xml.Name `xml:"package"`
	Manifest EpubManifest `xml:"manifest"`
}

func normExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if len(ext) > 0 && ext[0] == '.' {
		return ext[1:]
	}
	return ext
}

func walkDir(root string, d chan struct{}) (uint, error) {
	var (
		walk    func(string) error
		idx     int = -1
		dcnt    uint = 0
		modTime int64
	)

	defer close(d)
	walk = func(path string) error {
		dirEntries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		var dirs []FileInfo
		var files []FileInfo

		for _, entry := range dirEntries {
			fullPath := filepath.Join(path, entry.Name())
			if entry.IsDir() {
				if !cfg.cd {
					dirs = append(dirs, FileInfo{Path: fullPath, Name: "", isDir: true, isImage: false})
				}
			} else {
				ext := normExt(fullPath)
				if !slices.Contains(fileFormats, ext) { continue }

				if cfg.sa || cfg.sd {
					fi, _ := os.Stat(fullPath)
					modTime = fi.ModTime().Unix()
				}
				files = append(files, FileInfo{Path: fullPath, Name: entry.Name(), isDir: false, isImage: true, cPage: 0, modTime: modTime})
			}
		}

		if len(files) > 0 { // directories without relevant media are skipped
			if !cfg.flat {
				if cfg.sh {
					rand.Shuffle(len(files), func(i, j int) { files[i], files[j] = files[j], files[i] })
				} else if cfg.sd {
					sort.Slice(files, func(i, j int) bool { return files[i].modTime > files[j].modTime })
				} else if cfg.sa {
					sort.Slice(files, func(i, j int) bool { return files[i].modTime < files[j].modTime })
				} else {
					sort.Slice(files, func(i, j int) bool { return strings.ToLower(files[i].Path) < strings.ToLower(files[j].Path) })
				}
			}
			 idx++
			dcnt++
			fileInfos = append(fileInfos, FileInfo{ID: idx, Path: path, Name: "", isDir: true, isImage: false})

			for _, file := range files {
				idx++
				file.ID = idx
				fileInfos = append(fileInfos, file)
			}
		}

		sort.Slice(dirs, func(i, j int) bool {
			return strings.ToLower(dirs[i].Path) < strings.ToLower(dirs[j].Path)
		})
		for _, dir := range dirs {
			if err := walk(dir.Path); err != nil {
				continue // typically "Access denied."
			}
		}
		return nil
	}

	if err := walk(root); err != nil {
		return 0, err
	}

	if cfg.flat {
		if cfg.sh {
			rand.Shuffle(len(fileInfos), func(i, j int) {
				fileInfos[i], fileInfos[j] = fileInfos[j], fileInfos[i]
				fileInfos[i].ID, fileInfos[j].ID = fileInfos[j].ID, fileInfos[i].ID
			})
			goto _eoflat
		} else if cfg.sd {
			sort.Slice(fileInfos, func(i, j int) bool { return fileInfos[i].modTime > fileInfos[j].modTime })
		} else if cfg.sa {
			sort.Slice(fileInfos, func(i, j int) bool { return fileInfos[i].modTime < fileInfos[j].modTime })
		} else {
			sort.Slice(fileInfos, func(i, j int) bool { return strings.ToLower(fileInfos[i].Path) < strings.ToLower(fileInfos[j].Path) })
		}
		for i, _ := range fileInfos {
			fileInfos[i].ID = i
		}
_eoflat:
	}

	return dcnt, nil
}

func getEpubCoverImage(fp string) ([]byte, error) {
	var coverImageHref string

	var opf  EpubOPF
	var opfs EpubOPFSimple

	var opfFile        *zip.File
	var coverImageFile *zip.File

	hepubf, err := os.Open(fp)
	if err != nil {
		return nil, err
	}
	defer hepubf.Close()

	fileInfo, err := hepubf.Stat()
	if err != nil {
		return nil, err
	}
	zipReader, err := zip.NewReader(hepubf, fileInfo.Size())
	if err != nil {
		return nil, err
	}

	for _, f := range zipReader.File {
		ext := filepath.Ext(f.Name)
		if ext == ".opf" {
			opfFile = f
			break
		}
	}

	if opfFile == nil {
		return nil, err
	}

	hopf, err := opfFile.Open()
	if err != nil {
		return nil, err
	}
	defer hopf.Close()

	bval, _ := ioutil.ReadAll(hopf)

	// stripping <?xml..?> is required to Unmarshal properly
	cleanXML := string(bval)
	if strings.HasPrefix(cleanXML, "<?") {
		cleanXML = strings.SplitN(cleanXML, "?>", 2)[1]
		bval = []byte(cleanXML)
	}

	err = xml.Unmarshal(bval, &opf)
	if err != nil {
		xml.Unmarshal(bval, &opfs)
		opf = EpubOPF{
			Manifest: opfs.Manifest,
			Metadata: EpubMetadata{},
		}
	}

	if len(opf.Metadata.Meta) > 0 { // prio 1: metadata ref
		for _, meta := range opf.Metadata.Meta {
			if meta.Name == "cover" {
				coverImageHref = meta.Content
				break
			}
		}
	}

	if coverImageHref != "" {
		found := false
		for _, item := range opf.Manifest.Items {
			if item.ID == coverImageHref && strings.HasPrefix(item.MediaType, "image") {
				coverImageHref, _ = url.QueryUnescape(item.Href)
				found = true
				break
			}
		}
		if !found {
			coverImageHref = ""
		}
	}

	if coverImageHref == "" { // prio 2: naming convention
		for _, item := range opf.Manifest.Items {
			if !strings.HasPrefix(item.MediaType, "image") { continue }
			if strings.Contains(strings.ToLower(item.ID), "cover") || strings.Contains(strings.ToLower(item.Href), "cover") {
				coverImageHref, _ = url.QueryUnescape(item.Href)
				break
			}
		}
	}

	if coverImageHref == "" { // prio 3: first image item
		for _, item := range opf.Manifest.Items {
			if strings.HasPrefix(item.MediaType, "image") {
				coverImageHref, _ = url.QueryUnescape(item.Href)
				break
			}
		}
	}

	if coverImageHref != "" {
		for _, f := range zipReader.File {
			if strings.Contains(f.Name, coverImageHref) {
				coverImageFile = f
				break
			}
		}
	}

	if coverImageFile == nil {
		return nil, errors.New("404")
	}

	hcif, err := coverImageFile.Open()
	if err != nil {
		return nil, err
	}
	defer hcif.Close()

	imgBuf, err := ioutil.ReadAll(hcif)
	if err != nil {
		return nil, err
	}

	return imgBuf, nil
}

func getVipsPdfImage(pdfPath string, imageID int) ([]byte, error) {
	var img *vips.Image

	opts := vips.DefaultPdfloadOptions()
	opts.Page = 0

	img, err := vips.NewPdfload(pdfPath, opts)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	loadopts := &vips.ThumbnailImageOptions{ Height: 5000, }
	err = img.ThumbnailImage(int(cfg.width), loadopts)
	if err != nil {
		return nil, err
	}

	thumbnailBuf, err := img.JpegsaveBuffer(vipsJpegO)
	if err != nil {
		return nil, err
	}

	return thumbnailBuf, nil
}

func getFitzDocImage(fp string, imageID int) ([]byte, error) {
	var img image.Image

	// https://github.com/gen2brain/go-fitz/issues/4
	// locking all doc. ops is required
	doc, err := fitz.New(fp)
	if err != nil {
		return nil, err
	}

	defer func() {
		mu.Lock()
		doc.Close()
		mu.Unlock()
	}()

	mu.Lock()
	pages := doc.NumPage()
	mu.Unlock()

	// following a very basic approach assuming cover representation within the first 3 pages
	// much room for improvement handling non-trivial cases (via html parsing for example)
	for i, j, p, l := 0, 2, 0, 0; i < pages; i++ {
		mu.Lock()
		html, err := doc.HTML(i, false)
		mu.Unlock()
		if err != nil || i==j {
			if l > 10000 {
				mu.Lock()
				img, err = doc.Image(p)
				mu.Unlock()
				if err != nil {
					return nil, err
				}
				fileInfos[imageID].cPage = p
				break
			}

			mu.Lock()
			img, err = doc.Image(0)
			mu.Unlock()
			if err != nil {
				return nil, err
			}
			break
		} else {
			_len := len(html)
			if _len > l {
				l, p = _len, i
			}
		}
	}
	if img == nil {
		mu.Lock()
		img, err = doc.Image(0)
		mu.Unlock()
		if err != nil {
			return nil, err
		}
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	vi, _ := vips.NewImageFromMemory(rgba.Pix, bounds.Dx(), bounds.Dy(), 4)
	defer vi.Close()

	buf, _ := vi.JpegsaveBuffer(vipsJpegO)

	thumbnailBuf, err := getVipsFromBuffer(buf, true)
	if err != nil {
		return nil, err
	}

	return thumbnailBuf, nil
}

func getKindleCoverImage(fp string) ([]byte, error) {
	var (
		jpegStart = []byte{0xFF, 0xD8, 0xFF}
		jpegEnd   = []byte{0xFF, 0xD9}
		pngStart  = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		pngEnd    = []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}
	)

	data, err := ioutil.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	extractImage := func(startSignature, endSignature []byte) [][]byte {
		var images [][]byte
		startPos := 0

		for startPos < len(data) {
			startPosRel := bytes.Index(data[startPos:], startSignature)
			if startPosRel == -1 {
				break
			}
			startPos = startPos + startPosRel + len(startSignature)

			endPosRel := bytes.Index(data[startPos:], endSignature)
			if endPosRel == -1 {
				break
			}
			endPos := startPos + endPosRel + len(endSignature)

			image := data[startPos-len(startSignature):endPos]
			if len(image) > 0 {
				images = append(images, image)
			}

			startPos = endPos
		}
		return images
	}

	jpegImages := extractImage(jpegStart, jpegEnd)
	pngImages := extractImage(pngStart, pngEnd)

	var images [][]byte
	var imgBuf []byte

	images = append(images, jpegImages...)
	images = append(images, pngImages...)

	// naive approach for the time being
	for _, bytval := range images {   // extract first image of size
		if len(bytval) >= 50 * 1024 { // 50kb or larger
			imgBuf = bytval
			break
		}
	}

	if len(imgBuf) == 0 { // largest otherwise
		for _, bytval := range images {
			if len(bytval) > len(imgBuf) {
				imgBuf = bytval
			}
		}
	}

	if len(imgBuf) > 0 {
		return imgBuf, nil
	}

	return nil, errors.New("404")
}

func getVipsFromFile(fp string, resize bool) ([]byte, string, error) {
	var jmp bool = false

_Init:
	if !jmp {
		fi, _ := os.Stat(fp)
		if fi.Size() < thumbMinSize {
			var ct string = ""
			img, err := vips.NewImageFromFile(fp, nil)
			if err != nil {
				return nil, ct, err
			}
			defer img.Close()

			// handle special cases
			switch img.Format() {
				case "svg":
					ct = "image/svg+xml"
				case "jxl":
					jmp = true
					resize = true
					goto _Init
				case "jp2k", "tiff", "heif":
					jmp = true
					resize = false
					goto _Init
				default:
			}

			buf, err := os.ReadFile(fp)
			if err != nil {
				return nil, ct, err
			}
			return buf, ct, nil
		}
	}

	if resize {
		loadopts := &vips.ThumbnailOptions{ Height: 5000, }
		img, err := vips.NewThumbnail(fp, int(cfg.width), loadopts)
		if err != nil {
			return nil, "", err
		}
		defer img.Close()

		if img.Format() == "svg" {
			buf, _ := os.ReadFile(fp)
			return buf, "image/svg+xml", nil
		}

		thumbnailBuf, err := img.JpegsaveBuffer(vipsJpegO)
		if err != nil {
			return nil, "", err
		}

		return thumbnailBuf, "", nil
	}

	// sequential | https://www.libvips.org/API/8.17/enum.Access.html
	loadopts := &vips.LoadOptions{ Access: 1, }
	img, err := vips.NewImageFromFile(fp, loadopts)
	if err != nil {
		return nil, "", err
	}
	defer img.Close()

	if img.Format() == "svg" {
		buf, _ := os.ReadFile(fp)
		return buf, "image/svg+xml", nil
	}

	thumbnailBuf, err := img.JpegsaveBuffer(vipsJpegO)
	if err != nil {
		return nil, "", err
	}

	return thumbnailBuf, "", nil
}

func getVipsFromBuffer(buf []byte, resize bool) ([]byte, error) {

	if len(buf) < thumbMinSize {
		return buf, nil
	}

	if resize {
		loadopts := &vips.ThumbnailBufferOptions{ Height: 5000, }
		img, err := vips.NewThumbnailBuffer(buf, int(cfg.width), loadopts)
		if err != nil {
			return nil, err
		}
		defer img.Close()

		thumbnailBuf, err := img.JpegsaveBuffer(vipsJpegO)
		if err != nil {
			return nil, err
		}

		return thumbnailBuf, nil
	}

	img, err := vips.NewImageFromBuffer(buf, nil)
	if err != nil {
		return nil, err
	}
	defer img.Close()

	thumbnailBuf, err := img.JpegsaveBuffer(vipsJpegO)
	if err != nil {
		return nil, err
	}

	return thumbnailBuf, nil
}

func generateThumbnail(imageID int) ([]byte, string, error) {
	fp := fileInfos[imageID].Path
	ext := strings.ToLower(filepath.Ext(fp))

	var thumbnailBuf []byte
	var ct string = "image/jpeg"

	switch ext {
	case ".azw3": // consider similar for .mobi
		buf, err := getKindleCoverImage(fp)
		if err != nil {
			return nil, ct, err
		}
		thumbnailBuf, err = getVipsFromBuffer(buf, true)
		if err != nil {
			return nil, ct, err
		}

	case ".epub":
		buf, err := getEpubCoverImage(fp)
		if err != nil {
			thumbnailBuf, err = getFitzDocImage(fp, imageID)
			if err != nil {
				return nil, ct, err
			} else {
				return thumbnailBuf, ct, nil
			}
		}
		thumbnailBuf, err = getVipsFromBuffer(buf, true)
		if err != nil {
			return nil, ct, err
		}

	case ".mobi":
		var err error
		thumbnailBuf, err = getFitzDocImage(fp, imageID)
		if err != nil {
			return nil, ct, err
		}

	case ".pdf":
		var err error
		thumbnailBuf, err = getVipsPdfImage(fp, imageID)
		if err != nil {
			return nil, ct, err
		}

	// workarounds go here
	// bmp: unknown issue with vips or bindings
	case ".bmp":
		buf, err := os.ReadFile(fp)
		if err != nil {
			return nil, ct, err
		}

		fi, _ := os.Stat(fp)
		if fi.Size() < thumbMinSize {
			return thumbnailBuf, "image/bmp", nil
		}

		imgBmp, err := bmp.Decode(bytes.NewReader(buf))
		if err != nil {
			return nil, ct, err
		}

		var _buf bytes.Buffer
		if err := jpeg.Encode(&_buf, imgBmp, &jpeg.Options{Quality: 100}); err != nil {
			return nil, ct, err
		}
		thumbnailBuf, err = getVipsFromBuffer(_buf.Bytes(), true)
		if err != nil {
			return nil, ct, err
		}

	default:
		var err error
		thumbnailBuf, ct, err = getVipsFromFile(fp, true)
		if err != nil {
			return nil, ct, err
		}
	}

	return thumbnailBuf, ct, nil
}

func open(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "*bsd"
		cmd = "xdg-open"
	}
	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

func openWithDefaultApp(path string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", path)
	case "darwin":
		cmd = exec.Command("open", path)
	case "linux", "freebsd", "openbsd", "netbsd":
		cmd = exec.Command("xdg-open", path)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func contextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var data ContextData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	err = openWithDefaultApp(fileInfos[data.Id].Path)
	if err != nil {
		http.Error(w, "Error opening file: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "success"}`))
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	imageID, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/image/"))
	fp := fileInfos[imageID].Path

	// the default mode is trusting the file extension and serving the image as is
	// if the browser detects a load error then a single retry is attempted
	// ex. heic file.png won't be working per default
	var img image.Image
	var imgBuf []byte
	var file *os.File
	var err error
	var retryImage bool
	var jump bool

	if r.URL.Query().Get("retry") != "" {
		retryImage = true
	}

_Ext:
	ext := ".pdf"
	if !jump {
		ext = strings.ToLower(filepath.Ext(fp))
	}

	switch ext {
	case ".epub":
		imgBuf, err = getEpubCoverImage(fp)
		if err != nil { // retry via fitz
			jump = true
			goto _Ext
		}

	case ".azw3":
		imgBuf, err = getKindleCoverImage(fp)
		if err != nil {
			http.Error(w, "Unable to extract image: "+err.Error(), http.StatusInternalServerError)
			return
		}

	case ".pdf":
		var vi *vips.Image

		opts := vips.DefaultPdfloadOptions()
		opts.Page = fileInfos[imageID].cPage
		opts.Dpi = 144

		vi, err := vips.NewPdfload(fp, opts)
		if err != nil {
			http.Error(w, "Unable to open document: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer vi.Close()

		imgBuf, err = vi.JpegsaveBuffer(vipsJpegO)
		if err != nil {
			http.Error(w, "Unable to extract image: "+err.Error(), http.StatusInternalServerError)
			return
		}

	case ".mobi":
		doc, err := fitz.New(fp)
		if err != nil {
			http.Error(w, "Unable to open document: "+err.Error(), http.StatusInternalServerError)
			return
		}

		defer func() {
			mu.Lock()
			doc.Close()
			mu.Unlock()
		}()

		mu.Lock()
		img, err = doc.Image(fileInfos[imageID].cPage)
		mu.Unlock()
		if err != nil {
			http.Error(w, "Unable to extract image: "+err.Error(), http.StatusInternalServerError)
			return
		}

	default: // image
		if retryImage {
			imgBuf, _, err = getVipsFromFile(fp, false)
			if err != nil {
				http.Error(w, "Unable to serve image: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			file, err = os.Open(fp)
			if err != nil {
				http.Error(w, "Unable to open image: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer file.Close()
		}
	}

	// content type is determined by the browser, but we set it anyway
	switch ext {
	case ".jpg", ".jpeg", ".heic", ".jp2", ".jxl", ".pdf", ".epub", ".mobi", ".azw3":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".bmp":
		w.Header().Set("Content-Type", "image/bmp")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".avif":
		w.Header().Set("Content-Type", "image/avif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".tiff":
		w.Header().Set("Content-Type", "image/tiff")
	default:
		http.Error(w, "Unsupported image format", http.StatusUnsupportedMediaType)
	}

	if img != nil {
		jpeg.Encode(w, img, nil)
	} else if imgBuf != nil {
		w.Write(imgBuf)
	} else {
		_, err = io.Copy(w, file)
		if err != nil {
			http.Error(w, "Unable to serve image: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func thumbnailHandler(w http.ResponseWriter, r *http.Request) {
	imageID, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/thumbnail/"))

	buf, ct, err := generateThumbnail(imageID)
	if err != nil {
		http.Error(w, "Unable to generate thumbnail: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Write(buf)
}

func spin(d <-chan struct{}) {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)

	// cursor tuning
	fmt.Print("\033[?25l")
	defer func() {
		fmt.Print("\033[?25h")
		s.Stop()
	}()

	s.Suffix = " Indexing"
	s.Start()
	<-d
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Not enough arguments. Provide a search path.")
		os.Exit(0)
	}

	flag.BoolVar(&cfg.cd, "cd", false, "current directory only (no recursion)")
	flag.BoolVar(&cfg.flat, "f", false, "flatten directory tree")
	flag.StringVar(&cfg.ip, "i", "localhost", "bind ip; empty string \"\" for all")
	flag.BoolVar(&cfg.lsd, "lsd", true, "list all directories (including empty)")
	flag.BoolVar(&cfg.open, "o", false, "open webbrowser")
	flag.UintVar(&cfg.port, "p", 8989, "bind port")
	flag.BoolVar(&cfg.sa, "sa", false, "sort files by mod time asc")
	flag.BoolVar(&cfg.sd, "sd", false, "sort files by mod time desc")
	flag.BoolVar(&cfg.sh, "sh", false, "shuffle files")
	flag.BoolVar(&cfg.version, "v", false, "print version")
	flag.BoolVar(&cfg.verbose, "vv", false, "debug print version")
	flag.UintVar(&cfg.width, "w", 250, "thumbnail width in pixels")
	flag.BoolVar(&cfg.fit, "fit", true, "fit within viewport (vertical crop)")
	flag.Parse()

	// vips init
	vips.Startup(&vips.Config{})
	defer vips.Shutdown()

	if cfg.version {
		fmt.Println(version)
		os.Exit(0)
	}
	if cfg.verbose {
		fmt.Printf("thumbnailer: %v\n", version)
		fmt.Println("libvips:", vips.Version)
		fmt.Printf("FzVersion: %v\n", fitz.FzVersion)
		os.Exit(0)
	}

	if cfg.width != 250 {
		cssContent = strings.Replace(cssContent, "250px", fmt.Sprintf("%dpx", cfg.width), -1)
	}
	if cfg.flat { cfg.lsd = false }

	// spin while indexing
	   d := make(chan struct{})
	 res := make(chan uint)
	errc := make(chan error, 1)
	var dcnt uint

	go func() {
		dcnt, err := walkDir(os.Args[len(os.Args)-1], d)
		if err != nil {
			errc <- err
		}
		res <- dcnt
	}()

	spin(d)
	select {
		case dcnt = <-res:
		case err := <-errc:
			fmt.Println(err)
	}

	cssHidden := ""
	if !cfg.lsd || dcnt < 2 { cssHidden = " hidden" }

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(favicon)
	})
	http.HandleFunc("/thumbnail/", thumbnailHandler)
	http.HandleFunc("/image/", imageHandler)
	http.HandleFunc("/context/", contextHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		htmlContent := fmt.Sprintf(`<!doctype html>
<html>
<head>
<meta charset="utf-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>thumbnailer</title>
<link rel="icon" type="image/x-icon" href="/favicon.ico">
<style type="text/css">%s</style>
<script>%s</script>
</head>
<body data-width="%d" data-fit="%t">
<div class="menu%s" id="menu">&#9776;</div>
<ul class="menu-list" id="menuList"></ul>
<div class="menu-overlay" id="menuOverlay"></div>
<div id="lightbox">
	<div id="lightboxClose">&#x2716;</div>
	<div id="lightboxLoading"><p>Loading...</p></div>
	<img id="lightboxImage" src="" />
</div>
`, cssContent, jsContent, cfg.width, cfg.fit, cssHidden)
		w.Write([]byte(htmlContent))

		first := true
		var last bool

		for _, fileInfo := range fileInfos {
			if fileInfo.isImage {
				if first {
					w.Write([]byte(fmt.Sprintf("<ul class=\"flex\">")))
					first = false
				} else if last != fileInfo.isImage {
					w.Write([]byte(fmt.Sprintf("</ul><ul class=\"flex\">")))
				}
				last = fileInfo.isImage

				if cfg.lsd {
					w.Write([]byte(fmt.Sprintf("<li><img title=\"%s\" data-id=\"%d\" /><span class=\"name\">%s</span></li>", fileInfo.Name, fileInfo.ID, fileInfo.Name)))
				} else {
					w.Write([]byte(fmt.Sprintf("<li><img title=\"%s\" data-id=\"%d\" /><span class=\"name\">%s</span></li>", fileInfo.Path, fileInfo.ID, fileInfo.Name)))
				}
			} else if fileInfo.isDir && cfg.lsd {
				if first {
					w.Write([]byte(fmt.Sprintf("<ul class=\"stretch\">")))
					first = false
				} else if last != fileInfo.isImage {
					w.Write([]byte(fmt.Sprintf("</ul><ul class=\"stretch\">")))
				}
				last = fileInfo.isImage

				w.Write([]byte(fmt.Sprintf("<li><div class=\"dir-container\" id=\"%d\"><span>%s</span></div></li>", fileInfo.ID, fileInfo.Path)))
			}
		}
		w.Write([]byte("</ul></body></html>"))
	})

	ip := cfg.ip
	if cfg.ip == "" { ip = "localhost" }

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		if cfg.open {
			time.Sleep(250 * time.Millisecond)
			open(fmt.Sprintf("http://%s:%d", ip, cfg.port))
		}
		<-c
		os.Exit(0)
	}()

	fmt.Printf("Server running on http://%s:%d\nCtrl+c to exit\n", ip, cfg.port)
	http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.ip, cfg.port), nil)
}
