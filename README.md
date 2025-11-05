# thumbnailer cli

## [0.1.3] (2025-11-02)

Localhosted web app for displaying thumbnails in a grid layout.


### Supported media formats

* jpeg, png, gif, webp, bmp, heic, avif
* epub, pdf
* [basic via mupdf] mobi
* [experimental] azw3


### Manual

```
./thumbnailer [<flags>] <root path containing media>
	-h for help
```

* Refer to server info printed and open link in browser

* Click thumbnail to open lightbox

#### Key bindings
##### Lightbox
```
<-		previous

->		next

l		rotate left

r		rotate right

+/wheel	zoom in

-/wheel	zoom out

any		exit lightbox

mouseLeft	open image in new tab
```
##### Grid
```
mouseRight	open file in native app
```

### Future plans, pending features & issues

* search / index via menu

* support for djvu

* caching flag
	* for example using sqlite

* add more filetypes
	* https://github.com/h2non/filetype.py

* consider config defaults for ip:port and such

* color scheme; day/night mode

* [postponed] mobi/pdf: suppress error, warning and notice spam via mupdf lib
	* https://pkg.go.dev/github.com/gen2brain/go-fitz#section-readme

* [postponed] mobi appears to always miss the cover image
	* https://en.wikipedia.org/wiki/Comparison_of_e-book_formats#Microsoft_LIT

* [postponed] better support for Kindle formats (azw3)
	* https://github.com/kevinhendricks/KindleUnpack
	* https://stackoverflow.com/questions/5379565/kindle-periodical-format


### License

thumbnailer is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.
thumbnailer is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU General Public License for more details.
You should have received a copy of the GNU General Public License along with thumbnailer. If not, see <http://www.gnu.org/licenses/>
